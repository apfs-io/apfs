package fs

import "os"

// PathChecker used to check is this new path can be used to create new object
func PathChecker(abspath string) bool {
	_, err := os.Stat(abspath)
	// if list, _ := filepath.Glob(fullpath + "/*"); len(list) > 0 {
	// 	return nil, err
	// }
	return os.IsNotExist(err)
}
