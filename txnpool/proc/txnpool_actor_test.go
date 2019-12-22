// SPDX-License-Identifier: LGPL-3.0-or-later
// Copyright 2019 DNA Dev team
//
/*
 * Copyright (C) 2018 The ontology Authors
 * This file is part of The ontology library.
 *
 * The ontology is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The ontology is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with The ontology.  If not, see <http://www.gnu.org/licenses/>.
 */

package proc

import (
	"os"
	"testing"
	"time"

	"github.com/DNAProject/DNA/account"
	"github.com/DNAProject/DNA/common/config"
	"github.com/DNAProject/DNA/common/log"
	"github.com/DNAProject/DNA/consensus/vbft/config"
	"github.com/DNAProject/DNA/core/genesis"
	"github.com/DNAProject/DNA/core/ledger"
	"github.com/DNAProject/DNA/core/types"
	"github.com/DNAProject/DNA/errors"
	"github.com/DNAProject/DNA/events/message"
	tc "github.com/DNAProject/DNA/txnpool/common"
	vt "github.com/DNAProject/DNA/validator/types"
	"github.com/ontio/ontology-crypto/keypair"
	"github.com/stretchr/testify/assert"
)

var testGenesisConfig = &config.GenesisConfig{
	SeedList: []string{
		"localhost:20338",
		"localhost:20438",
		"localhost:20538",
		"localhost:20638",
		"localhost:20738"},
	ConsensusType: config.CONSENSUS_TYPE_VBFT,
	VBFT: &config.VBFTConfig{
		N:                    7,
		C:                    2,
		K:                    7,
		L:                    112,
		BlockMsgDelay:        10000,
		HashMsgDelay:         10000,
		PeerHandshakeTimeout: 10,
		MaxBlockChangeView:   120000,
		AdminOntID:           "did:dna:AdjfcJgwru2FD8kotCPvLDXYzRjqFjc9Tb",
		MinInitStake:         100000,
		VrfValue:             "",
		VrfProof:             "",
		Peers: []*config.VBFTPeerStakeInfo{
			{Index: 1},
			{Index: 2},
			{Index: 3},
			{Index: 4},
			{Index: 5},
			{Index: 6},
			{Index: 7},
		},
	},
	DBFT: &config.DBFTConfig{},
	SOLO: &config.SOLOConfig{},
}

func TestMain(m *testing.M) {
	log.InitLog(log.InfoLog, log.Stdout)
	var err error
	ledger.DefLedger, err = ledger.NewLedger(config.DEFAULT_DATA_DIR, 0)
	if err != nil {
		return
	}

	var bookkeepers []keypair.PublicKey
	testBookkeeperAccounts := make([]*account.Account, 0)
	for i := 0; i < 7; i++ {
		acc := account.NewAccount("")
		testBookkeeperAccounts = append(testBookkeeperAccounts, acc)
		bookkeepers = append(bookkeepers, acc.PublicKey)
	}

	config.DefConfig.Genesis = testGenesisConfig
	genesisConfig := config.DefConfig.Genesis

	// update peers in genesis
	for i, p := range genesisConfig.VBFT.Peers {
		if i < len(testBookkeeperAccounts) {
			p.PeerPubkey = vconfig.PubkeyID(testBookkeeperAccounts[i].PublicKey)
			p.Address = testBookkeeperAccounts[i].Address.ToBase58()
		}
	}

	block, err := genesis.BuildGenesisBlock(bookkeepers, genesisConfig)
	if err != nil {
		log.Fatalf("failed to build genesis block: %s", err)
	}
	err = ledger.DefLedger.Init(bookkeepers, block)
	if err != nil {
		log.Fatalf("failed to init default ledger: %s", err)
		return
	}

	m.Run()

	ledger.DefLedger.Close()
	os.RemoveAll(config.DEFAULT_DATA_DIR)
}

func TestTxActor(t *testing.T) {
	t.Log("Starting tx actor test")
	s := NewTxPoolServer(tc.MAX_WORKER_NUM, true, false)
	if s == nil {
		t.Error("Test case: new tx pool server failed")
		return
	}

	txActor := NewTxActor(s)
	txPid := startActor(txActor)
	if txPid == nil {
		t.Error("Test case: start tx actor failed")
		s.Stop()
		return
	}

	txReq := &tc.TxReq{
		Tx:     txn,
		Sender: tc.NilSender,
	}
	txPid.Tell(txReq)

	time.Sleep(1 * time.Second)

	future := txPid.RequestFuture(&tc.GetTxnReq{Hash: txn.Hash()}, 1*time.Second)
	result, err := future.Result()
	assert.Nil(t, err)
	rsp := (result).(*tc.GetTxnRsp)
	assert.Nil(t, rsp.Txn)

	future = txPid.RequestFuture(&tc.GetTxnStats{}, 2*time.Second)
	result, err = future.Result()
	assert.Nil(t, err)
	future = txPid.RequestFuture(&tc.CheckTxnReq{Hash: txn.Hash()}, 1*time.Second)
	result, err = future.Result()
	assert.Nil(t, err)

	future = txPid.RequestFuture(&tc.GetTxnStatusReq{Hash: txn.Hash()}, 1*time.Second)
	result, err = future.Result()
	assert.Nil(t, err)

	// Given the tx in the pool, test again
	txEntry := &tc.TXEntry{
		Tx:    txn,
		Attrs: []*tc.TXAttr{},
	}
	s.addTxList(txEntry)

	future = txPid.RequestFuture(&tc.GetTxnReq{Hash: txn.Hash()}, 1*time.Second)
	result, err = future.Result()
	assert.Nil(t, err)

	future = txPid.RequestFuture(&tc.GetTxnStats{}, 2*time.Second)
	result, err = future.Result()
	assert.Nil(t, err)
	future = txPid.RequestFuture(&tc.CheckTxnReq{Hash: txn.Hash()}, 1*time.Second)
	result, err = future.Result()
	assert.Nil(t, err)

	future = txPid.RequestFuture(&tc.GetTxnStatusReq{Hash: txn.Hash()}, 1*time.Second)
	result, err = future.Result()
	assert.Nil(t, err)

	txPid.Tell("test")
	s.Stop()
	t.Log("Ending tx actor test")
}

func TestTxPoolActor(t *testing.T) {
	t.Log("Starting tx pool actor test")
	s := NewTxPoolServer(tc.MAX_WORKER_NUM, true, false)
	if s == nil {
		t.Error("Test case: new tx pool server failed")
		return
	}

	txPoolActor := NewTxPoolActor(s)
	txPoolPid := startActor(txPoolActor)
	if txPoolPid == nil {
		t.Error("Test case: start tx actor failed")
		s.Stop()
		return
	}

	txEntry := &tc.TXEntry{
		Tx:    txn,
		Attrs: []*tc.TXAttr{},
	}

	retAttr := &tc.TXAttr{
		Height:  0,
		Type:    vt.Stateful,
		ErrCode: errors.ErrNoError,
	}
	txEntry.Attrs = append(txEntry.Attrs, retAttr)
	s.addTxList(txEntry)

	future := txPoolPid.RequestFuture(&tc.GetTxnPoolReq{ByCount: false}, 2*time.Second)
	result, err := future.Result()
	assert.Nil(t, err)
	rsp := (result).(*tc.GetTxnPoolRsp)
	assert.NotNil(t, rsp.TxnPool)

	future = txPoolPid.RequestFuture(&tc.GetPendingTxnReq{ByCount: false}, 2*time.Second)
	result, err = future.Result()
	assert.Nil(t, err)

	bk := &tc.VerifyBlockReq{
		Height: 0,
		Txs:    []*types.Transaction{txn},
	}
	future = txPoolPid.RequestFuture(bk, 10*time.Second)
	result, err = future.Result()
	assert.Nil(t, err)

	sbc := &message.SaveBlockCompleteMsg{}
	txPoolPid.Tell(sbc)

	s.Stop()
	t.Log("Ending tx pool actor test")
}

func TestVerifyRspActor(t *testing.T) {
	t.Log("Starting validator response actor test")
	s := NewTxPoolServer(tc.MAX_WORKER_NUM, true, false)
	if s == nil {
		t.Error("Test case: new tx pool server failed")
		return
	}

	validatorActor := NewVerifyRspActor(s)
	validatorPid := startActor(validatorActor)
	if validatorPid == nil {
		t.Error("Test case: start tx actor failed")
		s.Stop()
		return
	}

	validatorPid.Tell(txn)

	registerMsg := &vt.RegisterValidator{}
	validatorPid.Tell(registerMsg)

	unRegisterMsg := &vt.UnRegisterValidator{}
	validatorPid.Tell(unRegisterMsg)

	rsp := &vt.CheckResponse{}
	validatorPid.Tell(rsp)

	time.Sleep(1 * time.Second)
	s.Stop()
	t.Log("Ending validator response actor test")
}
