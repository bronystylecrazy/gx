package gx

import "flag"

func IsTestEnv() bool {
	return flag.Lookup("test.v") != nil
}
