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

package chaosd

import (
	"context"
	"strings"
	"syscall"

	"github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	"github.com/chaos-mesh/chaos-mesh/pkg/bpm"
	"github.com/google/uuid"
	"github.com/pingcap/errors"
	"github.com/pingcap/log"
	"github.com/rfyiamcool/go-timewheel"
	"github.com/shirou/gopsutil/process"
	"go.uber.org/zap"

	"github.com/chaos-mesh/chaosd/pkg/core"
)

func (s *Server) StressAttackScheduler(attack *core.StressCommand) (string, error) {
	uid := uuid.New().String()
	//create experiment
	if err := s.exp.Set(context.Background(), &core.Experiment{
		Uid:            uid,
		Status:         core.Created,
		Kind:           core.StressAttack,
		RecoverCommand: attack.String(),
	}); err != nil {
		return "", errors.WithStack(err)
	}

	if attack.CronInterval != 0 {
		task := s.tw.AddCron(attack.Duration + attack.CronInterval, func() {
			_, err := s.DoStressAttack(uid, attack)
			if err != nil {
				s.exp.Update(context.Background(), uid, core.Error, err.Error(), attack.String())
				log.Error("do stress experiment failed.", zap.Error(err))
			} else {
				s.exp.Update(context.Background(), uid, core.Running, "", attack.String())
				log.Info("running stress experiment.")
			}
		})
		taskMap.Store(uid, task)
	} else {
		_, err := s.DoStressAttack(uid, attack)
		if err != nil {
			s.exp.Update(context.Background(), uid, core.Error, err.Error(), attack.String())
			log.Error("do stress experiment failed.", zap.Error(err))
		} else {
			s.exp.Update(context.Background(), uid, core.Running, "", attack.String())
			log.Info("running stress experiment.")
		}
	}


	return uid, nil
}

// DoStressAttack will do stressAttack
func (s *Server) DoStressAttack(uid string, attack *core.StressCommand) (string, error) {
	var e error = nil
	s.tw.Add(attack.Duration, func() {
		err := s.DoRecoverStressAttack(uid, attack)
		if err != nil {
			status, _ := s.exp.GetStatus(uid)
			if status == core.Destroyed {
				return
			}
			s.exp.Update(context.Background(), uid, core.Error, err.Error(), attack.String())
			log.Error("do stress experiment recover failed.", zap.Error(err))
			e = err
		} else {
			if attack.CronInterval != 0 {
				s.exp.Update(context.Background(), uid, core.Waiting, "", attack.String())
				log.Info("waiting stress experiment.")
			} else {
				s.exp.Update(context.Background(), uid, core.Success, "", attack.String())
				log.Info("success stress experiment.")
			}

		}
	})
	if e != nil {
		return uid, e
	}

	stressors := &v1alpha1.Stressors{}
	if attack.Action == core.StressCPUAction {
		stressors.CPUStressor = &v1alpha1.CPUStressor{
			Stressor: v1alpha1.Stressor{
				Workers: attack.Workers,
			},
			Load:    &attack.Load,
			Options: attack.Options,
		}
	} else if attack.Action == core.StressMemAction {
		stressors.MemoryStressor = &v1alpha1.MemoryStressor{
			Stressor: v1alpha1.Stressor{
				Workers: attack.Workers,
			},
			Options: attack.Options,
		}
	}

	stressorsStr, err := stressors.Normalize()
	if err != nil {
		return "", err
	}
	log.Info("stressors normalize", zap.String("arguments", stressorsStr))

	cmd := bpm.DefaultProcessBuilder("stress-ng", strings.Fields(stressorsStr)...).Build()

	// Build will set SysProcAttr.Pdeathsig = syscall.SIGTERM, and so stress-ng will exit while chaosd exit
	// so reset it here
	cmd.Cmd.SysProcAttr = &syscall.SysProcAttr{}

	backgroundProcessManager := bpm.NewBackgroundProcessManager()
	err = backgroundProcessManager.StartProcess(cmd)
	if err != nil {
		return "", err
	}
	log.Info("Start stress-ng process successfully", zap.String("command", cmd.String()))

	attack.StressngPid = int32(cmd.Process.Pid)

	return uid, nil
}

func (s *Server) DoRecoverStressAttack(uid string, attack *core.StressCommand) error {
	proc, err := process.NewProcess(attack.StressngPid)
	if err != nil {
		return err
	}

	procName, err := proc.Name()
	if err != nil {
		return err
	}

	if !strings.Contains(procName, "stress-ng") {
		log.Warn("the process is not stress-ng, maybe it is killed by manual")
		return nil
	}

	if err := proc.Kill(); err != nil {
		log.Warn("the stress-ng process kill failed", zap.Error(err))
		return err
	}

	return nil
}

func (s *Server) StressAttack(attack *core.StressCommand) (string, error) {
	var err error

	uid := uuid.New().String()

	if err := s.exp.Set(context.Background(), &core.Experiment{
		Uid:            uid,
		Status:         core.Created,
		Kind:           core.StressAttack,
		RecoverCommand: attack.String(),
	}); err != nil {
		return "", errors.WithStack(err)
	}

	defer func() {
		if err != nil {
			if err := s.exp.Update(context.Background(), uid, core.Error, err.Error(), attack.String()); err != nil {
				log.Error("failed to update experiment", zap.Error(err))
			}
			return
		}

		// use the stressngPid as recover command, and will kill the pid when recover
		if err := s.exp.Update(context.Background(), uid, core.Success, "", attack.String()); err != nil {
			log.Error("failed to update experiment", zap.Error(err))
		}
	}()

	stressors := &v1alpha1.Stressors{}
	if attack.Action == core.StressCPUAction {
		stressors.CPUStressor = &v1alpha1.CPUStressor{
			Stressor: v1alpha1.Stressor{
				Workers: attack.Workers,
			},
			Load:    &attack.Load,
			Options: attack.Options,
		}
	} else if attack.Action == core.StressMemAction {
		stressors.MemoryStressor = &v1alpha1.MemoryStressor{
			Stressor: v1alpha1.Stressor{
				Workers: attack.Workers,
			},
			Options: attack.Options,
		}
	}

	stressorsStr, err := stressors.Normalize()
	if err != nil {
		return "", err
	}
	log.Info("stressors normalize", zap.String("arguments", stressorsStr))

	cmd := bpm.DefaultProcessBuilder("stress-ng", strings.Fields(stressorsStr)...).Build()

	// Build will set SysProcAttr.Pdeathsig = syscall.SIGTERM, and so stress-ng will exit while chaosd exit
	// so reset it here
	cmd.Cmd.SysProcAttr = &syscall.SysProcAttr{}

	backgroundProcessManager := bpm.NewBackgroundProcessManager()
	err = backgroundProcessManager.StartProcess(cmd)
	if err != nil {
		return "", err
	}
	log.Info("Start stress-ng process successfully", zap.String("command", cmd.String()))

	attack.StressngPid = int32(cmd.Process.Pid)

	return uid, nil
}

func (s *Server) RecoverStressAttack(uid string, attack *core.StressCommand) error {
	status, err := s.exp.GetStatus(uid)
	if err != nil {
		return err
	}

	if v, ok := taskMap.Load(uid); !ok && (status == core.Waiting || status == core.Running) {
		return errors.New("get task faild, uid: " + uid)
	} else {
		if task, ok := v.(*timewheel.Task); ok {
			log.Info("remove task success")
			s.tw.Remove(task)
		}
	}

	if status == core.Waiting {
		if err := s.exp.Update(context.Background(), uid, core.Destroyed, "", attack.String()); err != nil {
			return errors.WithStack(err)
		}
		return nil
	}

	proc, err := process.NewProcess(attack.StressngPid)
	if err != nil {
		return err
	}

	procName, err := proc.Name()
	if err != nil {
		return err
	}

	if !strings.Contains(procName, "stress-ng") {
		log.Warn("the process is not stress-ng, maybe it is killed by manual")
		return nil
	}

	if err := proc.Kill(); err != nil && status != core.Waiting {
		log.Error("the stress-ng process kill failed", zap.Error(err))
		return err
	}

	if err := s.exp.Update(context.Background(), uid, core.Destroyed, "", attack.String()); err != nil {
		return errors.WithStack(err)
	}

	return nil
}
