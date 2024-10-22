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

const (
	ErrPermissionedDenied    = "permission denied"
	ErrInvalidArgument       = "invalid argument"
	ErrNoSuchFileOrDirectory = "no such file or directory"
	ErrFileExists            = "file exists"
	ErrDirectoryNotEmpty     = "directory not empty"
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
	Command FtpCommand    `json:"command"`
	Success bool          `json:"success"`
	Code    int           `json:"code,omitempty"`
	Data    CommandResult `json:"data,omitempty"`
}

type CommandResult struct {
	Name     string          `json:"name,omitempty"`
	Type     string          `json:"type,omitempty"`
	Path     string          `json:"path,omitempty"`
	Size     int64           `json:"size,omitempty"`
	Children []CommandResult `json:"children,omitempty"`
	Message  string          `json:"message,omitempty"`
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
			ErrPermissionedDenied:    450,
			ErrInvalidArgument:       452,
			ErrNoSuchFileOrDirectory: 550,
			ErrFileExists:            552,
		},
	},
	Cwd: {
		Success: 250,
		Error: map[string]int{
			ErrPermissionedDenied:    450,
			ErrNoSuchFileOrDirectory: 550,
		},
	},
	Pwd: {
		Success: 250,
		Error: map[string]int{
			ErrPermissionedDenied:    450,
			ErrNoSuchFileOrDirectory: 550,
		},
	},
	Dele: {
		Success: 250,
		Error: map[string]int{
			ErrPermissionedDenied:    450,
			ErrInvalidArgument:       452,
			ErrNoSuchFileOrDirectory: 550,
		},
	},
	Rmd: {
		Success: 250,
		Error: map[string]int{
			ErrPermissionedDenied:    450,
			ErrInvalidArgument:       452,
			ErrNoSuchFileOrDirectory: 550,
			ErrDirectoryNotEmpty:     552,
		},
	},
	Mv: {
		Success: 250,
		Error: map[string]int{
			ErrPermissionedDenied:    450,
			ErrInvalidArgument:       452,
			ErrNoSuchFileOrDirectory: 550,
			ErrFileExists:            552,
		},
	},
	Cp: {
		Success: 250,
		Error: map[string]int{
			ErrPermissionedDenied:    450,
			ErrInvalidArgument:       452,
			ErrNoSuchFileOrDirectory: 550,
			ErrFileExists:            552,
		},
	},
}

func GetFtpErrorCode(command FtpCommand, result CommandResult) (CommandResult, int) {
	if codes, ok := returnCodes[command]; ok {
		for message, code := range codes.Error {
			if strings.Contains(result.Message, message) {
				return CommandResult{
					Message: message,
				}, code
			}
		}
	}
	// Default error code if not found
	return CommandResult{
		Message: result.Message,
	}, 550
}
