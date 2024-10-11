package runner

import "os"

type returnCode struct {
	Success int           `json:"success"`
	Error   map[error]int `json:"error"`
}

var returnCodes = map[string]returnCode{
	"list": {
		Success: 250,
		Error:   map[error]int{},
	},
	"mkd": {
		Success: 250,
		Error: map[error]int{
			os.ErrPermission: 450,
			os.ErrInvalid:    452,
			os.ErrNotExist:   550,
			os.ErrExist:      552,
		},
	},
	"cwd": {
		Success: 250,
		Error: map[error]int{
			os.ErrPermission: 450,
			os.ErrNotExist:   550,
		},
	},
	"pwd": {
		Success: 250,
		Error: map[error]int{
			os.ErrPermission: 450,
			os.ErrNotExist:   550,
		},
	},
	"dele": {
		Success: 250,
		Error: map[error]int{
			os.ErrPermission: 450,
			os.ErrInvalid:    452,
			os.ErrNotExist:   550,
		},
	},
	"rmd": {
		Success: 250,
		Error: map[error]int{
			os.ErrPermission: 450,
			os.ErrInvalid:    452,
			os.ErrNotExist:   550,
		},
	},
	"mv": {
		Success: 250,
		Error: map[error]int{
			os.ErrPermission: 450,
			os.ErrInvalid:    452,
			os.ErrNotExist:   550,
			os.ErrExist:      552,
		},
	},
	"cp": {
		Success: 250,
		Error: map[error]int{
			os.ErrPermission: 450,
			os.ErrInvalid:    452,
			os.ErrNotExist:   550,
			os.ErrExist:      552,
		},
	},
}

func GetFtpErrorCode(command string, err error) int {
	if codes, ok := returnCodes[command]; ok {
		if code, exists := codes.Error[err]; exists {
			return code
		}
	}
	// Default error code if not found
	return 550
}
