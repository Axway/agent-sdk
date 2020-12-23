package config

import "reflect"

// ValidateConfig - Validates the agent config
// Uses reflection to get the IConfigValidator interface on the config struct or
// struct variable.
// Makes call to ValidateCfg method except if the struct variable is of CentralConfig type
// as the validation for CentralConfig is already done during parseCentralConfig
func ValidateConfig(cfg interface{}) error {
	// Check if top level struct has Validate. If it does then call Validate
	// only at top level
	if cfg == nil {
		return nil
	}

	if objInterface, ok := cfg.(IConfigValidator); ok {
		err := objInterface.ValidateCfg()
		if err != nil {
			return err
		}
	}

	// If the parameter is of struct pointer, use indirection to get the
	// real value object
	v := reflect.ValueOf(cfg)
	if v.Kind() == reflect.Ptr {
		v = reflect.Indirect(v)
	}

	// Look for Validate method on struct properties and invoke it
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).CanInterface() {
			fieldInterface := v.Field(i).Interface()
			// Skip the property it is CentralConfig type as its already Validated
			// during parseCentralConfig

			if shouldValidateField(cfg, fieldInterface) {
				if objInterface, ok := fieldInterface.(IConfigValidator); ok {
					err := ValidateConfig(objInterface)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func shouldValidateField(cfg interface{}, fieldInterface interface{}) bool {
	_, isToplevelCentrlCfg := cfg.(CentralConfig)
	if isToplevelCentrlCfg {
		return true
	}

	_, isFieldCentralCfg := fieldInterface.(CentralConfig)
	if !isFieldCentralCfg {
		return true
	}
	return false
}
