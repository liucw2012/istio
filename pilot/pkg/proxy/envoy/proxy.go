// Copyright 2018 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package envoy

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/gogo/protobuf/types"

	meshconfig "istio.io/api/mesh/v1alpha1"
	"istio.io/istio/pilot/pkg/proxy"
	"istio.io/istio/pkg/bootstrap"
	"istio.io/pkg/env"
	"istio.io/pkg/log"
)

const (
	// epochFileTemplate is a template for the root config JSON
	epochFileTemplate = "envoy-rev%d.json"

	// drainFile is the location of the bootstrap config used for draining on istio-proxy termination
	drainFile = "/var/lib/istio/envoy/envoy_bootstrap_drain.json"
)

type envoy struct {
	config         meshconfig.ProxyConfig
	node           string
	extraArgs      []string
	pilotSAN       []string
	opts           map[string]interface{}
	nodeIPs        []string
	dnsRefreshRate string
}

// NewProxy creates an instance of the proxy control commands
func NewProxy(config meshconfig.ProxyConfig, node string, logLevel string,
	componentLogLevel string, pilotSAN []string, nodeIPs []string, dnsRefreshRate string, opts map[string]interface{}) proxy.Proxy {
	// inject tracing flag for higher levels
	var args []string
	if logLevel != "" {
		args = append(args, "-l", logLevel)
	}
	if componentLogLevel != "" {
		args = append(args, "--component-log-level", componentLogLevel)
	}

	return &envoy{
		config:         config,
		node:           node,
		extraArgs:      args,
		pilotSAN:       pilotSAN,
		nodeIPs:        nodeIPs,
		dnsRefreshRate: dnsRefreshRate,
		opts:           opts,
	}
}

func (e *envoy) args(fname string, epoch int, bootstrapConfig string) []string {
	proxyLocalAddressType := "v4"
	if isIPv6Proxy(e.nodeIPs) {
		proxyLocalAddressType = "v6"
	}
	startupArgs := []string{"-c", fname,
		"--restart-epoch", fmt.Sprint(epoch),
		"--drain-time-s", fmt.Sprint(int(convertDuration(e.config.DrainDuration) / time.Second)),
		"--parent-shutdown-time-s", fmt.Sprint(int(convertDuration(e.config.ParentShutdownDuration) / time.Second)),
		"--service-cluster", e.config.ServiceCluster,
		"--service-node", e.node,
		"--max-obj-name-len", fmt.Sprint(e.config.StatNameLength),
		"--local-address-ip-version", proxyLocalAddressType,
		"--allow-unknown-fields",
	}

	startupArgs = append(startupArgs, e.extraArgs...)

	if bootstrapConfig != "" {
		bytes, err := ioutil.ReadFile(bootstrapConfig)
		if err != nil {
			log.Warnf("Failed to read bootstrap override %s, %v", bootstrapConfig, err)
		} else {
			startupArgs = append(startupArgs, "--config-yaml", string(bytes))
		}
	}

	if e.config.Concurrency > 0 {
		startupArgs = append(startupArgs, "--concurrency", fmt.Sprint(e.config.Concurrency))
	}

	return startupArgs
}

var istioBootstrapOverrideVar = env.RegisterStringVar("ISTIO_BOOTSTRAP_OVERRIDE", "", "")

func (e *envoy) Run(config interface{}, epoch int, abort <-chan error) error {

	var fname string
	// Note: the cert checking still works, the generated file is updated if certs are changed.
	// We just don't save the generated file, but use a custom one instead. Pilot will keep
	// monitoring the certs and restart if the content of the certs changes.
	if len(e.config.CustomConfigFile) > 0 {
		// there is a custom configuration. Don't write our own config - but keep watching the certs.
		fname = e.config.CustomConfigFile
	} else if _, ok := config.(proxy.DrainConfig); ok {
		fname = drainFile
	} else {
		// 生成envoy的配置文件
		out, err := bootstrap.WriteBootstrap(
			&e.config, e.node, epoch, e.pilotSAN, e.opts, os.Environ(), e.nodeIPs, e.dnsRefreshRate)
		if err != nil {
			log.Errora("Failed to generate bootstrap config: ", err)
			os.Exit(1) // Prevent infinite loop attempting to write the file, let k8s/systemd report
			return err
		}
		fname = out
	}

	// spin up a new Envoy process
	// 构造envoy运行的配置文件
	args := e.args(fname, epoch, istioBootstrapOverrideVar.Get())
	log.Infof("Envoy command: %v", args)

	/* #nosec */
	// 运行
	cmd := exec.Command(e.config.BinaryPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() {
		// 等待运行结果
		done <- cmd.Wait()
	}()

	select {
	case err := <-abort:
		log.Warnf("Aborting epoch %d", epoch)
		if errKill := cmd.Process.Kill(); errKill != nil {
			log.Warnf("killing epoch %d caused an error %v", epoch, errKill)
		}
		return err
	case err := <-done:
		return err
	}
}

func (e *envoy) Cleanup(epoch int) {
	filePath := configFile(e.config.ConfigPath, epoch)
	// 删除此版本的配置文件
	if err := os.Remove(filePath); err != nil {
		log.Warnf("Failed to delete config file %s for %d, %v", filePath, epoch, err)
	}
}

func (e *envoy) Panic(epoch interface{}) {
	log.Error("cannot start the e with the desired configuration")
	if epochInt, ok := epoch.(int); ok {
		// print the failed config file
		// 打印所有失败的配置文件
		filePath := configFile(e.config.ConfigPath, epochInt)
		b, _ := ioutil.ReadFile(filePath)
		log.Errorf(string(b))
	}
	os.Exit(-1)
}

// convertDuration converts to golang duration and logs errors
func convertDuration(d *types.Duration) time.Duration {
	if d == nil {
		return 0
	}
	dur, err := types.DurationFromProto(d)
	if err != nil {
		log.Warnf("error converting duration %#v, using 0: %v", d, err)
	}
	return dur
}

func configFile(config string, epoch int) string {
	return path.Join(config, fmt.Sprintf(epochFileTemplate, epoch))
}

// isIPv6Proxy check the addresses slice and returns true for a valid IPv6 address
// for all other cases it returns false
func isIPv6Proxy(ipAddrs []string) bool {
	for i := 0; i < len(ipAddrs); i++ {
		addr := net.ParseIP(ipAddrs[i])
		if addr == nil {
			// Should not happen, invalid IP in proxy's IPAddresses slice should have been caught earlier,
			// skip it to prevent a panic.
			continue
		}
		if addr.To4() != nil {
			return false
		}
	}
	return true
}
