package runner

import "strings"

type returnCode struct {
	Success int            `json:"success"`
	Error   map[string]int `json:"error"`
}

var returnCodes = map[string]returnCode{
	"list": {
		Success: 250,
		Error:   map[string]int{},
	},
	"mkd": {
		Success: 250,
		Error: map[string]int{
			"permission denied":         450,
			"invalid argument":          452,
			"no such file or directory": 550,
			"file exists":               552,
		},
	},
	"cwd": {
		Success: 250,
		Error: map[string]int{
			"permission denied":         450,
			"no such file or directory": 550,
		},
	},
	"pwd": {
		Success: 250,
		Error: map[string]int{
			"permission denied":         450,
			"no such file or directory": 550,
		},
	},
	"dele": {
		Success: 250,
		Error: map[string]int{
			"permission denied":         450,
			"invalid argument":          452,
			"no such file or directory": 550,
		},
	},
	"rmd": {
		Success: 250,
		Error: map[string]int{
			"permission denied":         450,
			"invalid argument":          452,
			"no such file or directory": 550,
			"directory not empty":       552,
		},
	},
	"mv": {
		Success: 250,
		Error: map[string]int{
			"permission denied":         450,
			"invalid argument":          452,
			"no such file or directory": 550,
			"file exists":               552,
		},
	},
	"cp": {
		Success: 250,
		Error: map[string]int{
			"permission denied":         450,
			"invalid argument":          452,
			"no such file or directory": 550,
			"file exists":               552,
		},
	},
}

func GetFtpErrorCode(command string, result map[string]string) (map[string]string, int) {
	if codes, ok := returnCodes[command]; ok {
		for message, code := range codes.Error {
			if strings.Contains(result["message"], message) {
				return map[string]string{
					"message": message,
				}, code
			}
		}
	}
	// Default error code if not found
	return map[string]string{
		"message": result["message"],
	}, 550
}
