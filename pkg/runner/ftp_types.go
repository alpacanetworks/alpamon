package runner

import "strings"

type FtpCommand string

const (
	List FtpCommand = "list"
	Mkd  FtpCommand = "mkd"
	Cwd  FtpCommand = "cwd"
	Pwd  FtpCommand = "pwd"
	Dele FtpCommand = "dele"
	Rmd  FtpCommand = "rmd"
	Mv   FtpCommand = "mv"
	Cp   FtpCommand = "cp"
)

type FtpData struct {
	Path      string `json:"path,omitempty"`
	Depth     int    `json:"depth,omitempty"`
	Recursive bool   `json:"recursive,omitempty"`
	Src       string `json:"src,omitempty"`
	Dst       string `json:"dst,omitempty"`
}

type FtpContent struct {
	Command FtpCommand `json:"command"`
	Data    FtpData    `json:"data"`
}

type FtpResult struct {
	Command FtpCommand  `json:"command"`
	Success bool        `json:"success"`
	Code    int         `json:"code,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type FileInfo struct {
	Name     string     `json:"name"`
	Type     string     `json:"type"`
	Path     string     `json:"path"`
	Size     int64      `json:"size"`
	Children []FileInfo `json:"children,omitempty"`
	Message  string     `json:"message,omitempty"`
}

type CommandResult struct {
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
}

type returnCode struct {
	Success int            `json:"success"`
	Error   map[string]int `json:"error"`
}

var returnCodes = map[FtpCommand]returnCode{
	List: {
		Success: 250,
		Error:   map[string]int{},
	},
	Mkd: {
		Success: 250,
		Error: map[string]int{
			"permission denied":         450,
			"invalid argument":          452,
			"no such file or directory": 550,
			"file exists":               552,
		},
	},
	Cwd: {
		Success: 250,
		Error: map[string]int{
			"permission denied":         450,
			"no such file or directory": 550,
		},
	},
	Pwd: {
		Success: 250,
		Error: map[string]int{
			"permission denied":         450,
			"no such file or directory": 550,
		},
	},
	Dele: {
		Success: 250,
		Error: map[string]int{
			"permission denied":         450,
			"invalid argument":          452,
			"no such file or directory": 550,
		},
	},
	Rmd: {
		Success: 250,
		Error: map[string]int{
			"permission denied":         450,
			"invalid argument":          452,
			"no such file or directory": 550,
			"directory not empty":       552,
		},
	},
	Mv: {
		Success: 250,
		Error: map[string]int{
			"permission denied":         450,
			"invalid argument":          452,
			"no such file or directory": 550,
			"file exists":               552,
		},
	},
	Cp: {
		Success: 250,
		Error: map[string]int{
			"permission denied":         450,
			"invalid argument":          452,
			"no such file or directory": 550,
			"file exists":               552,
		},
	},
}

func GetFtpErrorCode(command FtpCommand, result map[string]string) (map[string]string, int) {
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
