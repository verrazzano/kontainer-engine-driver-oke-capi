// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"errors"
	"fmt"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg/k8s"
	"os"
	"strconv"
	"sync"

	"github.com/rancher/kontainer-engine/types"
	"github.com/verrazzano/kontainer-engine-driver-oke-capi/pkg"
	"go.uber.org/zap"
)

var wg = &sync.WaitGroup{}

func main() {
	if len(os.Args) < 2 || os.Args[1] == "" {
		panic(errors.New("no port provided"))
	}

	port, err := strconv.Atoi(os.Args[1])
	if err != nil {
		panic(fmt.Errorf("argument not parsable as int: %v", err))
	}

	k8s.MustSetKubeconfigFromEnv()
	logger := MustGetLogger()
	addr := make(chan string)
	go types.NewServer(&pkg.OKEDriver{
		Logger: logger,
	}, addr).ServeOrDie(fmt.Sprintf("127.0.0.1:%v", port))

	logger.Infof("+++ OKE CAPI driver up and running on at %v +++", <-addr)

	wg.Add(1)
	wg.Wait() // wait forever, we only exit if killed by parent process
}

func MustGetLogger() *zap.SugaredLogger {
	cfg := zap.NewProductionConfig()
	cfg.OutputPaths = []string{"stdout"}
	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	return logger.Sugar().With("kontainer-driver", "oke-capi")
}
