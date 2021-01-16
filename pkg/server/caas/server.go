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

package caas

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pingcap/log"
	"go.uber.org/zap"

	chaosmesh "github.com/chaos-mesh/chaos-mesh/api/v1alpha1"

	"github.com/chaos-mesh/chaosd/pkg/config"
	"github.com/chaos-mesh/chaosd/pkg/core"
	"github.com/chaos-mesh/chaosd/pkg/heartbeat"
	"github.com/chaos-mesh/chaosd/pkg/server/chaosd"
	"github.com/chaos-mesh/chaosd/pkg/server/utils"
	"github.com/chaos-mesh/chaosd/pkg/swaggerserver"
)

type httpServer struct {
	conf   *config.Config
	chaos  *chaosd.Server
	exp    core.ExperimentStore
	engine *gin.Engine

}

func NewServer(conf *config.Config, chaos *chaosd.Server, exp core.ExperimentStore) *httpServer {
	e := gin.Default()
	e.Use(utils.MWHandleErrors())
	return &httpServer{
		conf:   conf,
		chaos:  chaos,
		exp:    exp,
		engine: e,
	}
}

func Register(s *httpServer) {
	if s.conf.Platform != config.LocalPlatform {
		return
	}
	handler(s)
	go func() {
		addr := s.conf.Address()
		log.Debug("starting HTTP server", zap.String("address", addr))
		if err := s.engine.Run(addr); err != nil {
			log.Fatal("failed to start HTTP server", zap.Error(err))
		}
	}()

	// register
	go heartbeat.Register(s.conf)
}

func handler(s *httpServer) {
	api := s.engine.Group("/api")
	{
		api.GET("/swagger/*any", swaggerserver.Handler())
	}
	caas := api.Group("/caas")
	{
		caas.GET("/list", s.ListAttack)
		caas.POST("/stress", s.createStressAttack)
		caas.DELETE("/:uid", s.recoverAttack)
	}
}

// create
func (s *httpServer) createStressAttack(c *gin.Context) {
	var stressChaos chaosmesh.StressChaos
	var stressCommand core.StressCommand

	if err := c.ShouldBindJSON(&stressChaos); err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, utils.ErrInternalServer.WrapWithNoMessage(err))
		return
	}
	stressChaosJSON, _ := json.Marshal(stressChaos)
	log.Info("chaos data", zap.String("stress chaos json", string(stressChaosJSON)))

	attack := stressCommand.ConvertFromStressChaos(stressChaos)

	uid, err := s.chaos.StressAttackScheduler(attack)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, utils.ErrInternalServer.WrapWithNoMessage(err))
		return
	}
	c.JSON(http.StatusOK, utils.AttackSuccessResponse(uid))
}

// recover
func (s *httpServer) recoverAttack(c *gin.Context) {
	uid := c.Param("uid")

	err := utils.RecoverExp(s.exp, s.chaos, uid)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, utils.ErrInternalServer.WrapWithNoMessage(err))
		return
	}
	c.JSON(http.StatusOK, utils.RecoverSuccessResponse(uid))
}

// list
func (s *httpServer) ListAttack(c *gin.Context) {
	chaosList, err := s.exp.List(context.Background())
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, utils.ErrInternalServer.WrapWithNoMessage(err))
		return
	}

	chaosListJSON, _ := json.Marshal(chaosList)
	log.Info("chaos data", zap.String("list", string(chaosListJSON)))

	c.JSON(http.StatusOK, chaosList)
}
