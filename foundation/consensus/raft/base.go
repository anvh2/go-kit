package raft

import (
	"os"
	"time"

	"github.com/anvh2/go-kit/foundation/consensus/kv"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
)

type Config struct {
	RaftID              string
	RaftDir             string
	LogDir              string
	RaftAddr            string
	Timeout             time.Duration
	MaxPool             int
	RetainSnapshotCount int
}

type Raft struct {
	cfg  Config
	raft *raft.Raft
	kv   *kv.MemCached
}

func New(cfg Config, db *kv.MemCached) (*Raft, error) {
	config := raft.DefaultConfig()
	config.LogLevel = "INFO"
	config.LocalID = raft.ServerID(cfg.RaftID)

	transport, err := raft.NewTCPTransport(cfg.RaftAddr, nil, cfg.MaxPool, cfg.Timeout, os.Stderr)
	if err != nil {
		return nil, err
	}

	logStore, err := raftboltdb.NewBoltStore(cfg.LogDir)
	if err != nil {
		return nil, err
	}

	stableStore, err := kv.NewMemCached()
	if err != nil {
		return nil, err
	}

	snapshotStore, err := raft.NewFileSnapshotStore(cfg.RaftDir, cfg.RetainSnapshotCount, os.Stderr)
	if err != nil {
		return nil, err
	}

	raft, err := raft.NewRaft(config, nil, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return nil, err
	}

	return &Raft{
		kv:   db,
		cfg:  cfg,
		raft: raft,
	}, nil
}
