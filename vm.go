package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/go-jsonnet"
)

func getDebugger(cfg config) *jsonnet.Debugger {
	d := jsonnet.MakeDebugger()

	vm := d.GetVM()

	err := parseTLACode(cfg.tlaCode, vm)
	if err != nil {
		return nil
	}

	err = parseExtCode(cfg.extCode, vm)
	if err != nil {
		return nil
	}
	//parseTLACode(vm, cfg.tlaCode)
	d.SetVM(vm)

	return d
}

func parseTLACode(tlaCode map[string]interface{}, vm *jsonnet.VM) error {
	for k, v := range tlaCode {
		valueBytes, err := json.Marshal(v)
		if err != nil {
			return err
		}

		vm.TLACode(k, string(valueBytes))
	}
	return nil
}

func parseExtCode(extCode map[string]interface{}, vm *jsonnet.VM) error {
	for k, v := range extCode {
		valueBytes, err := json.Marshal(v)
		if err != nil {
			return err
		}

		vm.ExtCode(k, string(valueBytes))
	}
	return nil
}

func parseExtVars(unparsed interface{}) (map[string]string, error) {
	newVars, ok := unparsed.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unsupported settings value for ext_vars. expected json object. got: %T", unparsed)
	}

	extVars := make(map[string]string, len(newVars))
	for varKey, varValue := range newVars {
		vv, ok := varValue.(string)
		if !ok {
			return nil, fmt.Errorf("unsupported settings value for ext_vars.%s. expected string. got: %T", varKey, varValue)
		}
		extVars[varKey] = vv
	}
	return extVars, nil
}
