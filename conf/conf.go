// Package conf loads a service configuration from defaults, config.yaml and the environment.
package conf

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

// Carries key names only, never values.
type UnknownKeysError struct {
	Keys []string
}

func (e *UnknownKeysError) Error() string {
	return "unknown settings: " + strings.Join(e.Keys, ", ")
}

type options struct {
	strict bool
}

type Option func(*options)

// Strict rejects keys in config.yaml that cfg does not declare, so a typo cannot fall back to a default.
func Strict() Option {
	return func(o *options) { o.strict = true }
}

// Load fills cfg (a pointer to a struct holding its defaults) from ./config.yaml, then the environment.
// Settings are named by `mapstructure` tags; a setting's variable is its path upper-cased and joined by "__"
// (source.kind from SOURCE__KIND). Empty variables count as unset. Lists take "a,b" or JSON; maps and structs take JSON.
func Load(cfg any, opts ...Option) error {
	o := options{}
	for _, opt := range opts {
		opt(&o)
	}
	value := reflect.ValueOf(cfg)
	if value.Kind() != reflect.Pointer || value.Elem().Kind() != reflect.Struct {
		return errors.New("conf: cfg must be a pointer to a struct")
	}

	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "__"))
	v.AutomaticEnv()

	// Declares every setting, so the environment can set one that config.yaml does not.
	if err := v.MergeConfigMap(settings(value.Elem())); err != nil {
		return fmt.Errorf("conf: read the defaults: %w", err)
	}

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	var notFound viper.ConfigFileNotFoundError
	if err := v.MergeInConfig(); err != nil && !errors.As(err, &notFound) {
		return fmt.Errorf("conf: read config.yaml: %w", err)
	}

	var metadata mapstructure.Metadata
	err := v.Unmarshal(cfg, func(dc *mapstructure.DecoderConfig) {
		dc.Metadata = &metadata
		dc.DecodeHook = mapstructure.ComposeDecodeHookFunc(
			jsonStringHook,
			mapstructure.StringToTimeDurationHookFunc(),
			stringToSliceHook,
		)
	})
	if err != nil {
		return fmt.Errorf("conf: decode the settings: %w", err)
	}
	if o.strict && len(metadata.Unused) > 0 {
		unknown := slices.Clone(metadata.Unused)
		slices.Sort(unknown)
		return &UnknownKeysError{Keys: unknown}
	}
	return nil
}

func settings(value reflect.Value) map[string]any {
	out := map[string]any{}
	for i := range value.NumField() {
		field := value.Type().Field(i)
		if !field.IsExported() {
			continue
		}
		name, flags, _ := strings.Cut(field.Tag.Get("mapstructure"), ",")
		if name == "-" {
			continue
		}
		if name == "" {
			name = strings.ToLower(field.Name)
		}
		fieldValue := value.Field(i)
		if fieldValue.Kind() == reflect.Pointer && fieldValue.Type().Elem().Kind() == reflect.Struct {
			if fieldValue.IsNil() {
				fieldValue = reflect.New(fieldValue.Type().Elem())
			}
			fieldValue = fieldValue.Elem()
		}
		if fieldValue.Kind() != reflect.Struct {
			out[name] = fieldValue.Interface()
			continue
		}
		nested := settings(fieldValue)
		if strings.Contains(flags, "squash") {
			for k, v := range nested {
				out[k] = v
			}
			continue
		}
		out[name] = nested
	}
	return out
}

// mapstructure's own hook only handles []string.
func stringToSliceHook(from, to reflect.Type, data any) (any, error) {
	if from.Kind() != reflect.String || to.Kind() != reflect.Slice {
		return data, nil
	}
	raw := data.(string)
	if raw == "" {
		return []string{}, nil
	}
	return strings.Split(raw, ","), nil
}

func jsonStringHook(from, to reflect.Type, data any) (any, error) {
	if from.Kind() != reflect.String {
		return data, nil
	}
	if to.Kind() != reflect.Slice && to.Kind() != reflect.Map && to.Kind() != reflect.Struct {
		return data, nil
	}
	raw := strings.TrimSpace(data.(string))
	if !strings.HasPrefix(raw, "[") && !strings.HasPrefix(raw, "{") {
		return data, nil
	}
	out := reflect.New(to)
	if err := json.Unmarshal([]byte(raw), out.Interface()); err != nil {
		return nil, fmt.Errorf("decode %s from JSON: %w", to, err)
	}
	return out.Elem().Interface(), nil
}
