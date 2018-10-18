package logp

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

var (
	ToStderr          *bool
	DebugSelectorsStr *string
)

type Logging struct {
	Selectors []string
	Files     *FileRotator
	ToSyslog  *bool `config:"to_syslog"`
	ToFiles   *bool `config:"to_files"`
	JSON      bool  `config:"json"`
	Level     string
}

func HandleFlags(name string) error {
	level := _log.level
	selectors := strings.Split(*DebugSelectorsStr, ",")
	debugSelectors, debugAll := parseSelectors(selectors)
	if debugAll || len(debugSelectors) > 0 {
		level = LOG_DEBUG
	}

	// flags are handled before config file is read => log to stderr for now
	_log.level = level
	_log.toStderr = true
	_log.logger = log.New(os.Stderr, name, stderrLogFlags)
	_log.selectors = debugSelectors
	_log.debugAllSelectors = debugAll

	return nil
}

// Init combines the configuration from config with the command line
// flags to initialize the Logging systems. After calling this function,
// standard output is always enabled. You can make it respect the command
// line flag with a later SetStderr call.
func Init(name string, config *Logging) error {
	// reset settings from HandleFlags
	_log = logger{
		JSON: config.JSON,
	}

	logLevel, err := getLogLevel(config)
	if err != nil {
		return err
	}

	debugSelectors := config.Selectors
	if logLevel == LOG_DEBUG {
		if len(debugSelectors) == 0 {
			debugSelectors = []string{"*"}
		}
	}

	if DebugSelectorsStr == nil {
		d := ""
		DebugSelectorsStr = &d
	}
	if len(*DebugSelectorsStr) > 0 {
		debugSelectors = strings.Split(*DebugSelectorsStr, ",")
		logLevel = LOG_DEBUG
	}

	// default log location is in the logs path
	defaultFilePath := Resolve(Logs, "")

	var toSyslog, toFiles bool
	if config.ToSyslog != nil {
		toSyslog = *config.ToSyslog
	} else {
		toSyslog = false
	}
	if config.ToFiles != nil {
		toFiles = *config.ToFiles
	} else {
		toFiles = true
	}

	if ToStderr == nil {
		t := false
		ToStderr = &t
	}
	// ToStderr disables logging to syslog/files
	if *ToStderr {
		toSyslog = false
		toFiles = false
	}

	LogInit(Priority(logLevel), "", toSyslog, *ToStderr, debugSelectors)
	if len(debugSelectors) > 0 {
		config.Selectors = debugSelectors
	}

	if toFiles {
		if config.Files == nil {
			config.Files = &FileRotator{
				Path: defaultFilePath,
				Name: name,
			}
		} else {
			if config.Files.Path == "" {
				config.Files.Path = defaultFilePath
			}

			if config.Files.Name == "" {
				config.Files.Name = name
			}
		}

		err := SetToFile(true, config.Files)
		if err != nil {
			return err
		}
	}

	if IsDebug("stdlog") {
		// disable standard logging by default (this is sometimes
		// used by libraries and we don't want their logs to spam ours)
		log.SetOutput(ioutil.Discard)
	}

	// Disable stderr logging if requested by cmdline flag
	SetStderr()
	return nil
}

func SetStderr() {
	if !*ToStderr {
		SetToStderr(false, "")
		Debug("log", "Disable stderr logging")
	}
}

func getLogLevel(config *Logging) (Priority, error) {
	if config == nil || config.Level == "" {
		return LOG_INFO, nil
	}

	levels := map[string]Priority{
		"critical": LOG_CRIT,
		"error":    LOG_ERR,
		"warning":  LOG_WARNING,
		"info":     LOG_INFO,
		"debug":    LOG_DEBUG,
	}

	level, ok := levels[strings.ToLower(config.Level)]
	if !ok {
		return 0, fmt.Errorf("unknown log level: %v", config.Level)
	}
	return level, nil
}
