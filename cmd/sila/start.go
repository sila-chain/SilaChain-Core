// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//

package main

import (
	"fmt"

	"github.com/sila-org/sila/internal/telemetry/tracesetup"
	"time"

	"github.com/sila-org/sila/accounts"
	"github.com/sila-org/sila/cmd/utils"
	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/eth/downloader"
	"github.com/sila-org/sila/ethclient"
	"github.com/sila-org/sila/log"
	"github.com/sila-org/sila/node"
	"github.com/urfave/cli/v2"
)

// StartExecutionNode starts the node and shared execution runtime services.
func StartExecutionNode(ctx *cli.Context, stack *node.Node, isConsole bool) {
	utils.StartNode(ctx, stack, isConsole)

	if ctx.IsSet(utils.UnlockedAccountFlag.Name) {
		log.Warn(`The "unlock" flag has been deprecated and has no effect`)
	}

	startWalletLifecycle(stack)

	if ctx.Bool(utils.ExitWhenSyncedFlag.Name) {
		startSyncExitLifecycle(stack)
	}
}

func startWalletLifecycle(stack *node.Node) {
	events := make(chan accounts.WalletEvent, 16)
	stack.AccountManager().Subscribe(events)

	rpcClient := stack.Attach()
	ethClient := ethclient.NewClient(rpcClient)

	go func() {
		for _, wallet := range stack.AccountManager().Wallets() {
			if err := wallet.Open(""); err != nil {
				log.Warn("Failed to open wallet", "url", wallet.URL(), "err", err)
			}
		}

		for event := range events {
			switch event.Kind {
			case accounts.WalletArrived:
				if err := event.Wallet.Open(""); err != nil {
					log.Warn("New wallet appeared, failed to open", "url", event.Wallet.URL(), "err", err)
				}

			case accounts.WalletOpened:
				status, _ := event.Wallet.Status()

				log.Info(
					"New wallet appeared",
					"url", event.Wallet.URL(),
					"status", status,
				)

				var derivationPaths []accounts.DerivationPath

				if event.Wallet.URL().Scheme == "ledger" {
					derivationPaths = append(
						derivationPaths,
						accounts.LegacyLedgerBaseDerivationPath,
					)
				}

				derivationPaths = append(
					derivationPaths,
					accounts.DefaultBaseDerivationPath,
				)

				event.Wallet.SelfDerive(
					derivationPaths,
					ethClient,
				)

			case accounts.WalletDropped:
				log.Info(
					"Old wallet dropped",
					"url", event.Wallet.URL(),
				)

				event.Wallet.Close()
			}
		}
	}()
}

func startSyncExitLifecycle(stack *node.Node) {
	go func() {
		sub := stack.EventMux().Subscribe(downloader.DoneEvent{})
		defer sub.Unsubscribe()

		for {
			event := <-sub.Chan()

			if event == nil {
				continue
			}

			done, ok := event.Data.(downloader.DoneEvent)
			if !ok {
				continue
			}

			timestamp := time.Unix(int64(done.Latest.Time), 0)

			if time.Since(timestamp) < 10*time.Minute {
				log.Info(
					"Synchronisation completed",
					"latestnum", done.Latest.Number,
					"latesthash", done.Latest.Hash(),
					"age", common.PrettyAge(timestamp),
				)

				stack.Close()
			}
		}
	}()
}

// SetupTelemetry sets up OpenTelemetry reporting if enabled.
func SetupTelemetry(cfg node.OpenTelemetryConfig, stack *node.Node) error {
	return tracesetup.SetupTelemetry(cfg, stack)
}

// SyncTargetFromContext parses the optional sync target hash from CLI context.
func SyncTargetFromContext(ctx *cli.Context) (common.Hash, error) {
	var synctarget common.Hash
	if ctx.IsSet(utils.SyncTargetFlag.Name) {
		target := ctx.String(utils.SyncTargetFlag.Name)
		if !common.IsHexHash(target) {
			return common.Hash{}, fmt.Errorf("sync target hash is not a valid hex hash: %s", target)
		}
		synctarget = common.HexToHash(target)
	}
	return synctarget, nil
}
