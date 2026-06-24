// Copyright 2015 The sila Authors
// This file is part of the sila library.
//
// The sila library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The sila library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the sila library. If not, see <http://www.gnu.org/licenses/>.

// Package silaweb3ext contains Sila-specific silaWeb3.js extensions.
package silaweb3ext

var Modules = map[string]string{
	"admin":  AdminJs,
	"clique": CliqueJs,
	"debug":  DebugJs,
	"sila":   SilaJs,
	"miner":  MinerJs,
	"net":    NetJs,
	"rpc":    RpcJs,
	"txpool": TxpoolJs,
	"dev":    DevJs,
}

const CliqueJs = `
silaWeb3._extend({
	property: 'clique',
	methods: [
		new silaWeb3._extend.Method({
			name: 'getSnapshot',
			call: 'clique_getSnapshot',
			params: 1,
			inputFormatter: [silaWeb3._extend.formatters.inputBlockNumberFormatter]
		}),
		new silaWeb3._extend.Method({
			name: 'getSnapshotAtHash',
			call: 'clique_getSnapshotAtHash',
			params: 1
		}),
		new silaWeb3._extend.Method({
			name: 'getSigners',
			call: 'clique_getSigners',
			params: 1,
			inputFormatter: [silaWeb3._extend.formatters.inputBlockNumberFormatter]
		}),
		new silaWeb3._extend.Method({
			name: 'getSignersAtHash',
			call: 'clique_getSignersAtHash',
			params: 1
		}),
		new silaWeb3._extend.Method({
			name: 'propose',
			call: 'clique_propose',
			params: 2
		}),
		new silaWeb3._extend.Method({
			name: 'discard',
			call: 'clique_discard',
			params: 1
		}),
		new silaWeb3._extend.Method({
			name: 'status',
			call: 'clique_status',
			params: 0
		}),
		new silaWeb3._extend.Method({
			name: 'getSigner',
			call: 'clique_getSigner',
			params: 1,
			inputFormatter: [null]
		}),
	],
	properties: [
		new silaWeb3._extend.Property({
			name: 'proposals',
			getter: 'clique_proposals'
		}),
	]
});
`

const AdminJs = `
silaWeb3._extend({
	property: 'admin',
	methods: [
		new silaWeb3._extend.Method({
			name: 'addPeer',
			call: 'admin_addPeer',
			params: 1
		}),
		new silaWeb3._extend.Method({
			name: 'removePeer',
			call: 'admin_removePeer',
			params: 1
		}),
		new silaWeb3._extend.Method({
			name: 'addTrustedPeer',
			call: 'admin_addTrustedPeer',
			params: 1
		}),
		new silaWeb3._extend.Method({
			name: 'removeTrustedPeer',
			call: 'admin_removeTrustedPeer',
			params: 1
		}),
		new silaWeb3._extend.Method({
			name: 'exportChain',
			call: 'admin_exportChain',
			params: 3,
			inputFormatter: [null, null, null]
		}),
		new silaWeb3._extend.Method({
			name: 'importChain',
			call: 'admin_importChain',
			params: 1
		}),
		new silaWeb3._extend.Method({
			name: 'sleepBlocks',
			call: 'admin_sleepBlocks',
			params: 2
		}),
		new silaWeb3._extend.Method({
			name: 'startHTTP',
			call: 'admin_startHTTP',
			params: 5,
			inputFormatter: [null, null, null, null, null]
		}),
		new silaWeb3._extend.Method({
			name: 'stopHTTP',
			call: 'admin_stopHTTP'
		}),
		// This method is deprecated.
		new silaWeb3._extend.Method({
			name: 'startRPC',
			call: 'admin_startRPC',
			params: 5,
			inputFormatter: [null, null, null, null, null]
		}),
		// This method is deprecated.
		new silaWeb3._extend.Method({
			name: 'stopRPC',
			call: 'admin_stopRPC'
		}),
		new silaWeb3._extend.Method({
			name: 'startWS',
			call: 'admin_startWS',
			params: 4,
			inputFormatter: [null, null, null, null]
		}),
		new silaWeb3._extend.Method({
			name: 'stopWS',
			call: 'admin_stopWS'
		}),
		new silaWeb3._extend.Method({
			name: 'clearTxpool',
			call: 'debug_clearTxpool',
			params: 0
		}),
	],
	properties: [
		new silaWeb3._extend.Property({
			name: 'nodeInfo',
			getter: 'admin_nodeInfo'
		}),
		new silaWeb3._extend.Property({
			name: 'peers',
			getter: 'admin_peers'
		}),
		new silaWeb3._extend.Property({
			name: 'datadir',
			getter: 'admin_datadir'
		}),
	]
});
`

const DebugJs = `
silaWeb3._extend({
	property: 'debug',
	methods: [
		new silaWeb3._extend.Method({
			name: 'accountRange',
			call: 'debug_accountRange',
			params: 6,
			inputFormatter: [silaWeb3._extend.formatters.inputDefaultBlockNumberFormatter, null, null, null, null, null],
		}),
		new silaWeb3._extend.Method({
			name: 'printBlock',
			call: 'debug_printBlock',
			params: 1,
			outputFormatter: console.log
		}),
		new silaWeb3._extend.Method({
			name: 'getRawHeader',
			call: 'debug_getRawHeader',
			params: 1
		}),
		new silaWeb3._extend.Method({
			name: 'getRawBlock',
			call: 'debug_getRawBlock',
			params: 1
		}),
		new silaWeb3._extend.Method({
			name: 'getRawReceipts',
			call: 'debug_getRawReceipts',
			params: 1
		}),
		new silaWeb3._extend.Method({
			name: 'getRawTransaction',
			call: 'debug_getRawTransaction',
			params: 1
		}),
		new silaWeb3._extend.Method({
			name: 'setHead',
			call: 'debug_setHead',
			params: 1
		}),
		new silaWeb3._extend.Method({
			name: 'dumpBlock',
			call: 'debug_dumpBlock',
			params: 1,
			inputFormatter: [silaWeb3._extend.formatters.inputBlockNumberFormatter]
		}),
		new silaWeb3._extend.Method({
			name: 'chaindbProperty',
			call: 'debug_chaindbProperty',
			outputFormatter: console.log
		}),
		new silaWeb3._extend.Method({
			name: 'chaindbCompact',
			call: 'debug_chaindbCompact',
		}),
		new silaWeb3._extend.Method({
			name: 'verbosity',
			call: 'debug_verbosity',
			params: 1
		}),
		new silaWeb3._extend.Method({
			name: 'vmodule',
			call: 'debug_vmodule',
			params: 1
		}),
		new silaWeb3._extend.Method({
			name: 'stacks',
			call: 'debug_stacks',
			params: 1,
			inputFormatter: [null],
			outputFormatter: console.log
		}),
		new silaWeb3._extend.Method({
			name: 'freeOSMemory',
			call: 'debug_freeOSMemory',
			params: 0,
		}),
		new silaWeb3._extend.Method({
			name: 'setGCPercent',
			call: 'debug_setGCPercent',
			params: 1,
		}),
		new silaWeb3._extend.Method({
			name: 'setMemoryLimit',
			call: 'debug_setMemoryLimit',
			params: 1,
		}),
		new silaWeb3._extend.Method({
			name: 'memStats',
			call: 'debug_memStats',
			params: 0,
		}),
		new silaWeb3._extend.Method({
			name: 'gcStats',
			call: 'debug_gcStats',
			params: 0,
		}),
		new silaWeb3._extend.Method({
			name: 'cpuProfile',
			call: 'debug_cpuProfile',
			params: 2
		}),
		new silaWeb3._extend.Method({
			name: 'startCPUProfile',
			call: 'debug_startCPUProfile',
			params: 1
		}),
		new silaWeb3._extend.Method({
			name: 'stopCPUProfile',
			call: 'debug_stopCPUProfile',
			params: 0
		}),
		new silaWeb3._extend.Method({
			name: 'goTrace',
			call: 'debug_goTrace',
			params: 2
		}),
		new silaWeb3._extend.Method({
			name: 'startGoTrace',
			call: 'debug_startGoTrace',
			params: 1
		}),
		new silaWeb3._extend.Method({
			name: 'stopGoTrace',
			call: 'debug_stopGoTrace',
			params: 0
		}),
		new silaWeb3._extend.Method({
			name: 'blockProfile',
			call: 'debug_blockProfile',
			params: 2
		}),
		new silaWeb3._extend.Method({
			name: 'setBlockProfileRate',
			call: 'debug_setBlockProfileRate',
			params: 1
		}),
		new silaWeb3._extend.Method({
			name: 'writeBlockProfile',
			call: 'debug_writeBlockProfile',
			params: 1
		}),
		new silaWeb3._extend.Method({
			name: 'mutexProfile',
			call: 'debug_mutexProfile',
			params: 2
		}),
		new silaWeb3._extend.Method({
			name: 'setMutexProfileFraction',
			call: 'debug_setMutexProfileFraction',
			params: 1
		}),
		new silaWeb3._extend.Method({
			name: 'writeMutexProfile',
			call: 'debug_writeMutexProfile',
			params: 1
		}),
		new silaWeb3._extend.Method({
			name: 'writeMemProfile',
			call: 'debug_writeMemProfile',
			params: 1
		}),
		new silaWeb3._extend.Method({
			name: 'traceBlock',
			call: 'debug_traceBlock',
			params: 2,
			inputFormatter: [null, null]
		}),
		new silaWeb3._extend.Method({
			name: 'traceBlockFromFile',
			call: 'debug_traceBlockFromFile',
			params: 2,
			inputFormatter: [null, null]
		}),
		new silaWeb3._extend.Method({
			name: 'traceBadBlock',
			call: 'debug_traceBadBlock',
			params: 1,
			inputFormatter: [null]
		}),
		new silaWeb3._extend.Method({
			name: 'standardTraceBadBlockToFile',
			call: 'debug_standardTraceBadBlockToFile',
			params: 2,
			inputFormatter: [null, null]
		}),
		new silaWeb3._extend.Method({
			name: 'intermediateRoots',
			call: 'debug_intermediateRoots',
			params: 2,
			inputFormatter: [null, null]
		}),
		new silaWeb3._extend.Method({
			name: 'standardTraceBlockToFile',
			call: 'debug_standardTraceBlockToFile',
			params: 2,
			inputFormatter: [null, null]
		}),
		new silaWeb3._extend.Method({
			name: 'traceBlockByNumber',
			call: 'debug_traceBlockByNumber',
			params: 2,
			inputFormatter: [silaWeb3._extend.formatters.inputBlockNumberFormatter, null]
		}),
		new silaWeb3._extend.Method({
			name: 'traceBlockByHash',
			call: 'debug_traceBlockByHash',
			params: 2,
			inputFormatter: [null, null]
		}),
		new silaWeb3._extend.Method({
			name: 'traceTransaction',
			call: 'debug_traceTransaction',
			params: 2,
			inputFormatter: [null, null]
		}),
		new silaWeb3._extend.Method({
			name: 'traceCall',
			call: 'debug_traceCall',
			params: 3,
			inputFormatter: [null, null, null]
		}),
		new silaWeb3._extend.Method({
			name: 'preimage',
			call: 'debug_preimage',
			params: 1,
			inputFormatter: [null]
		}),
		new silaWeb3._extend.Method({
			name: 'getBadBlocks',
			call: 'debug_getBadBlocks',
			params: 0,
		}),
		new silaWeb3._extend.Method({
			name: 'storageRangeAt',
			call: 'debug_storageRangeAt',
			params: 5,
		}),
		new silaWeb3._extend.Method({
			name: 'getModifiedAccountsByNumber',
			call: 'debug_getModifiedAccountsByNumber',
			params: 2,
			inputFormatter: [null, null],
		}),
		new silaWeb3._extend.Method({
			name: 'getModifiedAccountsByHash',
			call: 'debug_getModifiedAccountsByHash',
			params: 2,
			inputFormatter:[null, null],
		}),
		new silaWeb3._extend.Method({
			name: 'getAccessibleState',
			call: 'debug_getAccessibleState',
			params: 2,
			inputFormatter:[silaWeb3._extend.formatters.inputBlockNumberFormatter, silaWeb3._extend.formatters.inputBlockNumberFormatter],
		}),
		new silaWeb3._extend.Method({
			name: 'dbGet',
			call: 'debug_dbGet',
			params: 1
		}),
		new silaWeb3._extend.Method({
			name: 'dbAncient',
			call: 'debug_dbAncient',
			params: 2
		}),
		new silaWeb3._extend.Method({
			name: 'dbAncients',
			call: 'debug_dbAncients',
			params: 0
		}),
		new silaWeb3._extend.Method({
			name: 'setTrieFlushInterval',
			call: 'debug_setTrieFlushInterval',
			params: 1
		}),
		new silaWeb3._extend.Method({
			name: 'getTrieFlushInterval',
			call: 'debug_getTrieFlushInterval',
			params: 0
		}),
		new silaWeb3._extend.Method({
			name: 'sync',
			call: 'debug_sync',
			params: 1
		}),
		new silaWeb3._extend.Method({
			name: 'stateSize',
			call: 'debug_stateSize',
			params: 1,
			inputFormatter: [null],
		}),
		new silaWeb3._extend.Method({
			name: 'executionWitness',
			call: 'debug_executionWitness',
			params: 1,
			inputFormatter: [null],
		}),
	],
	properties: []
});
`

const SilaJs = `
silaWeb3._extend({
	property: 'sila',
	methods: [
		new silaWeb3._extend.Method({
			name: 'chainId',
			call: 'sila_chainId',
			params: 0
		}),
		new silaWeb3._extend.Method({
			name: 'sign',
			call: 'sila_sign',
			params: 2,
			inputFormatter: [silaWeb3._extend.formatters.inputAddressFormatter, null]
		}),
		new silaWeb3._extend.Method({
			name: 'resend',
			call: 'sila_resend',
			params: 3,
			inputFormatter: [silaWeb3._extend.formatters.inputTransactionFormatter, silaWeb3._extend.utils.fromDecimal, silaWeb3._extend.utils.fromDecimal]
		}),
		new silaWeb3._extend.Method({
			name: 'signTransaction',
			call: 'sila_signTransaction',
			params: 1,
			inputFormatter: [silaWeb3._extend.formatters.inputTransactionFormatter]
		}),
		new silaWeb3._extend.Method({
			name: 'estimateGas',
			call: 'sila_estimateGas',
			params: 4,
			inputFormatter: [silaWeb3._extend.formatters.inputCallFormatter, silaWeb3._extend.formatters.inputBlockNumberFormatter, null, null],
			outputFormatter: silaWeb3._extend.utils.toDecimal
		}),
		new silaWeb3._extend.Method({
			name: 'submitTransaction',
			call: 'sila_submitTransaction',
			params: 1,
			inputFormatter: [silaWeb3._extend.formatters.inputTransactionFormatter]
		}),
		new silaWeb3._extend.Method({
			name: 'fillTransaction',
			call: 'sila_fillTransaction',
			params: 1,
			inputFormatter: [silaWeb3._extend.formatters.inputTransactionFormatter]
		}),
		new silaWeb3._extend.Method({
			name: 'getHeaderByNumber',
			call: 'sila_getHeaderByNumber',
			params: 1,
			inputFormatter: [silaWeb3._extend.formatters.inputBlockNumberFormatter]
		}),
		new silaWeb3._extend.Method({
			name: 'getHeaderByHash',
			call: 'sila_getHeaderByHash',
			params: 1
		}),
		new silaWeb3._extend.Method({
			name: 'getBlockByNumber',
			call: 'sila_getBlockByNumber',
			params: 2,
			inputFormatter: [silaWeb3._extend.formatters.inputBlockNumberFormatter, function (val) { return !!val; }]
		}),
		new silaWeb3._extend.Method({
			name: 'getBlockByHash',
			call: 'sila_getBlockByHash',
			params: 2,
			inputFormatter: [null, function (val) { return !!val; }]
		}),
		new silaWeb3._extend.Method({
			name: 'getRawTransaction',
			call: 'sila_getRawTransactionByHash',
			params: 1
		}),
		new silaWeb3._extend.Method({
			name: 'getRawTransactionFromBlock',
			call: function(args) {
				return (silaWeb3._extend.utils.isString(args[0]) && args[0].indexOf('0x') === 0) ? 'sila_getRawTransactionByBlockHashAndIndex' : 'sila_getRawTransactionByBlockNumberAndIndex';
			},
			params: 2,
			inputFormatter: [silaWeb3._extend.formatters.inputBlockNumberFormatter, silaWeb3._extend.utils.toHex]
		}),
		new silaWeb3._extend.Method({
			name: 'getProof',
			call: 'sila_getProof',
			params: 3,
			inputFormatter: [silaWeb3._extend.formatters.inputAddressFormatter, null, silaWeb3._extend.formatters.inputDefaultBlockNumberFormatter]
		}),
		new silaWeb3._extend.Method({
			name: 'getStorageValues',
			call: 'sila_getStorageValues',
			params: 2,
			inputFormatter: [null, silaWeb3._extend.formatters.inputDefaultBlockNumberFormatter]
		}),
		new silaWeb3._extend.Method({
			name: 'createAccessList',
			call: 'sila_createAccessList',
			params: 2,
			inputFormatter: [null, silaWeb3._extend.formatters.inputBlockNumberFormatter],
		}),
		new silaWeb3._extend.Method({
			name: 'feeHistory',
			call: 'sila_feeHistory',
			params: 3,
			inputFormatter: [null, silaWeb3._extend.formatters.inputBlockNumberFormatter, null]
		}),
		new silaWeb3._extend.Method({
			name: 'getLogs',
			call: 'sila_getLogs',
			params: 1,
		}),
		new silaWeb3._extend.Method({
			name: 'call',
			call: 'sila_call',
			params: 4,
			inputFormatter: [silaWeb3._extend.formatters.inputCallFormatter, silaWeb3._extend.formatters.inputDefaultBlockNumberFormatter, null, null],
		}),
		new silaWeb3._extend.Method({
			name: 'simulateV1',
			call: 'sila_simulateV1',
			params: 2,
			inputFormatter: [null, silaWeb3._extend.formatters.inputDefaultBlockNumberFormatter],
		}),
		new silaWeb3._extend.Method({
			name: 'getBlockReceipts',
			call: 'sila_getBlockReceipts',
			params: 1,
		}),
		new silaWeb3._extend.Method({
			name: 'config',
			call: 'sila_config',
			params: 0,
		}),
		new silaWeb3._extend.Method({
			name: 'capabilities',
			call: 'sila_capabilities',
			params: 0,
		})
	],
	properties: [
		new silaWeb3._extend.Property({
			name: 'pendingTransactions',
			getter: 'sila_pendingTransactions',
			outputFormatter: function(txs) {
				var formatted = [];
				for (var i = 0; i < txs.length; i++) {
					formatted.push(silaWeb3._extend.formatters.outputTransactionFormatter(txs[i]));
					formatted[i].blockHash = null;
				}
				return formatted;
			}
		}),
		new silaWeb3._extend.Property({
			name: 'maxPriorityFeePerGas',
			getter: 'sila_maxPriorityFeePerGas',
			outputFormatter: silaWeb3._extend.utils.toBigNumber
		}),
	]
});
`

const MinerJs = `
silaWeb3._extend({
	property: 'miner',
	methods: [
		new silaWeb3._extend.Method({
			name: 'setExtra',
			call: 'miner_setExtra',
			params: 1
		}),
		new silaWeb3._extend.Method({
			name: 'setGasPrice',
			call: 'miner_setGasPrice',
			params: 1,
			inputFormatter: [silaWeb3._extend.utils.fromDecimal]
		}),
		new silaWeb3._extend.Method({
			name: 'setGasLimit',
			call: 'miner_setGasLimit',
			params: 1,
			inputFormatter: [silaWeb3._extend.utils.fromDecimal]
		}),
	],
	properties: []
});
`

const NetJs = `
silaWeb3._extend({
	property: 'net',
	methods: [],
	properties: [
		new silaWeb3._extend.Property({
			name: 'version',
			getter: 'net_version'
		}),
	]
});
`

const RpcJs = `
silaWeb3._extend({
	property: 'rpc',
	methods: [],
	properties: [
		new silaWeb3._extend.Property({
			name: 'modules',
			getter: 'rpc_modules'
		}),
	]
});
`

const TxpoolJs = `
silaWeb3._extend({
	property: 'txpool',
	methods: [],
	properties:
	[
		new silaWeb3._extend.Property({
			name: 'content',
			getter: 'txpool_content'
		}),
		new silaWeb3._extend.Property({
			name: 'inspect',
			getter: 'txpool_inspect'
		}),
		new silaWeb3._extend.Property({
			name: 'status',
			getter: 'txpool_status',
			outputFormatter: function(status) {
				status.pending = silaWeb3._extend.utils.toDecimal(status.pending);
				status.queued = silaWeb3._extend.utils.toDecimal(status.queued);
				return status;
			}
		}),
		new silaWeb3._extend.Method({
			name: 'contentFrom',
			call: 'txpool_contentFrom',
			params: 1,
		}),
	]
});
`

const DevJs = `
silaWeb3._extend({
	property: 'dev',
	methods:
	[
		new silaWeb3._extend.Method({
			name: 'addWithdrawal',
			call: 'dev_addWithdrawal',
			params: 1
		}),
		new silaWeb3._extend.Method({
			name: 'setFeeRecipient',
			call: 'dev_setFeeRecipient',
			params: 1
		}),
	],
});
`
