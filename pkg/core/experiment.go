// Copyright 2020 Chaos Mesh Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package core

import (
	"context"
	"time"

	timewheel "github.com/rfyiamcool/go-timewheel"
)

const (
	Created   = "created"
	Waiting   = "Waiting"
	Running   = "Running"
	Success   = "success"
	Error     = "error"
	Destroyed = "destroyed"
	Revoked   = "revoked"
)

const (
	ProcessAttack = "process"
	NetworkAttack = "network"
	StressAttack  = "stress"
)

// ExperimentStore defines operations for working with experiments
type ExperimentStore interface {
	List(ctx context.Context) ([]*Experiment, error)
	ListByConditions(ctx context.Context, conds *SearchCommand) ([]*Experiment, error)
	ListByStatus(ctx context.Context, status string) ([]*Experiment, error)
	FindByUid(ctx context.Context, uid string) (*Experiment, error)
	Set(ctx context.Context, exp *Experiment) error
	Update(ctx context.Context, uid, status, msg string, command string) error
	SetTask(uid string, task *timewheel.Task) error
	GetTask(uid string) (*timewheel.Task, error)
}

// Experiment represents an experiment instance.
type Experiment struct {
	ID        uint      `gorm:"primary_key" json:"id"`
	Uid       string    `gorm:"index:uid" json:"uid"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Status    string    `json:"status"`
	Message   string    `json:"error"`
	// TODO: need to improve
	Kind           string `json:"kind"`
	Action         string `json:"action"`
	RecoverCommand string `json:"recover_command"`
	Task           *timewheel.Task
}
