// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package silaexec

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"reflect"
	"unicode"

	"github.com/naoina/toml"

	"github.com/sila-org/sila/cmd/utils"
	"github.com/sila-org/sila/log"
)

var deprecatedConfigFields = map[string]bool{
	"ethconfig.Config.EVMInterpreter":          true,
	"ethconfig.Config.EWASMInterpreter":        true,
	"ethconfig.Config.TrieCleanCacheJournal":   true,
	"ethconfig.Config.TrieCleanCacheRejournal": true,
	"ethconfig.Config.LightServ":               true,
	"ethconfig.Config.LightIngress":            true,
	"ethconfig.Config.LightEgress":             true,
	"ethconfig.Config.LightPeers":              true,
	"ethconfig.Config.LightNoPrune":            true,
	"ethconfig.Config.LightNoSyncServe":        true,
}

var ConfigTOMLSettings = toml.Config{
	NormFieldName: func(rt reflect.Type, key string) string {
		return key
	},
	FieldToKey: func(rt reflect.Type, field string) string {
		return field
	},
	MissingField: func(rt reflect.Type, field string) error {
		id := fmt.Sprintf("%s.%s", rt.String(), field)
		if deprecatedConfigFields[id] {
			log.Warn(fmt.Sprintf("Config field '%s' is deprecated and won't have any effect.", id))
			return nil
		}
		var link string
		if unicode.IsUpper(rune(rt.Name()[0])) && rt.PkgPath() != "main" {
			link = fmt.Sprintf(", see https://godoc.org/%s#%s for available fields", rt.PkgPath(), rt.Name())
		}
		return fmt.Errorf("field '%s' is not defined in %s%s", field, rt.String(), link)
	},
}

func LoadConfig(file string, cfg any) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	err = ConfigTOMLSettings.NewDecoder(bufio.NewReader(f)).Decode(cfg)
	if _, ok := err.(*toml.LineError); ok {
		err = errors.New(file + ", " + err.Error())
	}
	return err
}

func LoadConfigOrFatal(file string, cfg any) {
	if file == "" {
		return
	}
	if err := LoadConfig(file, cfg); err != nil {
		utils.Fatalf("%v", err)
	}
}
