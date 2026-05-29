// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The SilaChain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the SilaChain library. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"crypto/rand"
	"math/big"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"context"

	"github.com/sila-org/sila/internal/version"
	"github.com/sila-org/sila/rpc"
)

const (
	ipcAPIs  = "admin:1.0 debug:1.0 miner:1.0 rpc:1.0 sila:1.0 silaEngine:1.0 silaNet:1.0 silaWeb3:1.0 testing:1.0 txpool:1.0"
	httpAPIs = "rpc:1.0 sila:1.0 silaNet:1.0 silaWeb3:1.0"
)

// spawns sila with the given command line args, using a set of flags to minimise
// memory and disk IO. If the args don't set --datadir, the
// child g gets a temporary data directory.
func runMinimalSila(t *testing.T, args ...string) *testSila {
	// --holesky to make the 'writing genesis to disk' faster (no accounts)
	// --networkid=1337 to avoid cache bump
	// --syncmode=full to avoid allocating fast sync bloom
	allArgs := []string{"--holesky", "--networkid", "1337", "--authrpc.port", "0", "--syncmode=full", "--port", "0",
		"--nat", "none", "--nodiscover", "--maxpeers", "0", "--cache", "64",
		"--datadir.minfreedisk", "0"}
	return runSila(t, append(allArgs, args...)...)
}

// Tests that a node embedded within a console can be started up properly and
// then terminated by closing the input stream.
func TestConsoleWelcome(t *testing.T) {
	t.Parallel()
	coinbase := "0x8605cdbbdb6d264aa742e77020dcbc58fcdce182"

	// Start a sila console, make sure it's cleaned up and terminate the console
	sila := runMinimalSila(t, "--miner.pending.feeRecipient", coinbase, "console")

	// Gather all the infos the welcome message needs to contain
	sila.SetTemplateFunc("goos", func() string { return runtime.GOOS })
	sila.SetTemplateFunc("goarch", func() string { return runtime.GOARCH })
	sila.SetTemplateFunc("gover", runtime.Version)
	sila.SetTemplateFunc("silaver", func() string { return version.WithCommit("", "") })
	sila.SetTemplateFunc("niltime", func() string {
		return time.Unix(1695902100, 0).Format("Mon Jan 02 2006 15:04:05 GMT-0700 (MST)")
	})
	sila.SetTemplateFunc("apis", func() string { return ipcAPIs })

	// Verify the actual welcome message to the required template
	sila.Expect(`
Welcome to the Sila JavaScript console!

instance: Sila/v{{silaver}}/{{goos}}-{{goarch}}/{{gover}}
at block: 0 ({{niltime}})
 datadir: {{.Datadir}}
 modules: {{apis}}

To exit, press ctrl-d or type exit
> {{.InputLine "exit"}}
`)
	sila.ExpectExit()
}

// Tests that a console can be attached to a running node via various means.
func TestAttachWelcome(t *testing.T) {
	var (
		ipc      string
		httpPort string
		wsPort   string
	)
	// Configure the instance for IPC attachment
	if runtime.GOOS == "windows" {
		ipc = `\\.\pipe\sila` + strconv.Itoa(trulyRandInt(100000, 999999))
	} else {
		ipc = filepath.Join(t.TempDir(), "sila.ipc")
	}
	// And HTTP + WS attachment
	p := trulyRandInt(1024, 65533) // Yeah, sometimes this will fail, sorry :P
	httpPort = strconv.Itoa(p)
	wsPort = strconv.Itoa(p + 1)
	sila := runMinimalSila(t, "--miner.pending.feeRecipient", "0x8605cdbbdb6d264aa742e77020dcbc58fcdce182",
		"--ipcpath", ipc,
		"--http", "--http.port", httpPort,
		"--ws", "--ws.port", wsPort)
	defer sila.Kill()
	t.Run("ipc", func(t *testing.T) {
		waitForEndpoint(t, ipc, 2*time.Minute)
		testAttachWelcome(t, sila, "ipc:"+ipc, ipcAPIs)
	})
	t.Run("http", func(t *testing.T) {
		endpoint := "http://127.0.0.1:" + httpPort
		waitForEndpoint(t, endpoint, 2*time.Minute)
		testAttachWelcome(t, sila, endpoint, httpAPIs)
	})
	t.Run("ws", func(t *testing.T) {
		endpoint := "ws://127.0.0.1:" + wsPort
		waitForEndpoint(t, endpoint, 2*time.Minute)
		testAttachWelcome(t, sila, endpoint, httpAPIs)
	})
}

func TestSilaOnlyRPCModules(t *testing.T) {
	t.Parallel()

	var ipc string
	if runtime.GOOS == "windows" {
		ipc = `\\.\pipe\sila` + strconv.Itoa(trulyRandInt(100000, 999999))
	} else {
		ipc = filepath.Join(t.TempDir(), "sila.ipc")
	}

	sila := runMinimalSila(t,
		"--ipcpath", ipc,
		"--http",
		"--http.api", "sila,silaNet,silaWeb3",
		"--ws",
		"--ws.api", "sila,silaNet,silaWeb3",
	)
	defer sila.Kill()

	waitForEndpoint(t, ipc, 2*time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := rpc.DialContext(ctx, ipc)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	modules := make(map[string]string)
	if err := client.CallContext(ctx, &modules, "rpc_modules"); err != nil {
		t.Fatal(err)
	}

	for _, namespace := range []string{"sila", "silaNet", "silaWeb3"} {
		if modules[namespace] != "1.0" {
			t.Fatalf("missing namespace %q in modules: %v", namespace, modules)
		}
	}

	for _, namespace := range []string{"web3", "net"} {
		if _, ok := modules[namespace]; ok {
			t.Fatalf("unexpected namespace %q exposed in Sila-only modules: %v", namespace, modules)
		}
	}
}
func testAttachWelcome(t *testing.T, sila *testSila, endpoint, apis string) {
	// Attach to a running sila node and terminate immediately
	attach := runSila(t, "attach", endpoint)
	defer attach.ExpectExit()
	attach.CloseStdin()

	// Gather all the infos the welcome message needs to contain
	attach.SetTemplateFunc("goos", func() string { return runtime.GOOS })
	attach.SetTemplateFunc("goarch", func() string { return runtime.GOARCH })
	attach.SetTemplateFunc("gover", runtime.Version)
	attach.SetTemplateFunc("silaver", func() string { return version.WithCommit("", "") })
	attach.SetTemplateFunc("niltime", func() string {
		return time.Unix(1695902100, 0).Format("Mon Jan 02 2006 15:04:05 GMT-0700 (MST)")
	})
	attach.SetTemplateFunc("ipc", func() bool { return strings.HasPrefix(endpoint, "ipc") })
	attach.SetTemplateFunc("datadir", func() string { return sila.Datadir })
	attach.SetTemplateFunc("apis", func() string { return apis })

	// Verify the actual welcome message to the required template
	attach.Expect(`
Welcome to the Sila JavaScript console!
{{if ipc}}
instance: Sila/v{{silaver}}/{{goos}}-{{goarch}}/{{gover}}
at block: 0 ({{niltime}})
 datadir: {{datadir}}{{end}}
 modules: {{apis}}

To exit, press ctrl-d or type exit
> {{.InputLine "exit" }}
`)
	attach.ExpectExit()
}

// trulyRandInt generates a crypto random integer used by the console tests to
// not clash network ports with other tests running concurrently.
func trulyRandInt(lo, hi int) int {
	num, _ := rand.Int(rand.Reader, big.NewInt(int64(hi-lo)))
	return int(num.Int64()) + lo
}
