package utils

import (
	"fmt"
)

func Get[T any](fname, def string, fMap func(string) (T, bool), key ...string) (T, error) {
	if len(key) == 0 {
		cli, is := fMap(def)
		if is {
			return cli, nil
		}
		return cli, fmt.Errorf("[get] %s %s not found", fname, def)
	}
	cli, is := fMap(key[0])
	if is {
		return cli, nil
	}
	return cli, fmt.Errorf("[get] %s %s not found", fname, key[0])
}

func Init[T any, C any](fname, def string, keys []string, cfgs map[string]C, fNew func(cfg C) (T, error), fDef func(T)) (map[string]T, error) {
	dMap := make(map[string]T)
	if len(keys) != 0 {
		keys = append(keys, def)
		for _, key := range keys {
			_, is := dMap[key]
			if is {
				continue
			}
			cfg, is := cfgs[key]
			if !is {
				return nil, fmt.Errorf("%s %s not found", fname, key)
			}
			cli, err := fNew(cfg)
			if err != nil {
				return nil, fmt.Errorf("init %s %s init error: %v", fname, key, err)
			}
			dMap[key] = cli
			if key == def {
				fDef(cli)
			}
		}
		return dMap, nil
	}
	for name, cfg := range cfgs {
		cli, err := fNew(cfg)
		if err != nil {
			return nil, fmt.Errorf("init %s %s init error: %v", fname, name, err)
		}
		dMap[name] = cli
		if name == def {
			fDef(cli)
		}
	}
	return dMap, nil
}
