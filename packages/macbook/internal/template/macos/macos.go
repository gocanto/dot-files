package macos

type Setting struct {
	Domain string
	Key    string
	Args   []string
}

func Settings() []Setting {
	return []Setting{
		{"NSGlobalDomain", "AppleInterfaceStyle", []string{"-string", "Dark"}},
		{"NSGlobalDomain", "AppleShowAllExtensions", []string{"-bool", "true"}},
		{"NSGlobalDomain", "ApplePressAndHoldEnabled", []string{"-bool", "false"}},
		{"NSGlobalDomain", "NSAutomaticDashSubstitutionEnabled", []string{"-bool", "false"}},
		{"NSGlobalDomain", "NSAutomaticQuoteSubstitutionEnabled", []string{"-bool", "false"}},
		{"NSGlobalDomain", "NSAutomaticPeriodSubstitutionEnabled", []string{"-bool", "false"}},
		{"NSGlobalDomain", "NSNavPanelExpandedStateForSaveMode", []string{"-bool", "true"}},
		{"NSGlobalDomain", "PMPrintingExpandedStateForPrint", []string{"-bool", "true"}},
		{"com.apple.finder", "AppleShowAllFiles", []string{"-bool", "true"}},
		{"com.apple.finder", "ShowPathbar", []string{"-bool", "true"}},
		{"com.apple.finder", "ShowStatusBar", []string{"-bool", "true"}},
		{"com.apple.finder", "FXPreferredViewStyle", []string{"-string", "Nlsv"}},
		{"com.apple.finder", "_FXShowPosixPathInTitle", []string{"-bool", "true"}},
		{"com.apple.dock", "autohide", []string{"-bool", "true"}},
		{"com.apple.dock", "mineffect", []string{"-string", "scale"}},
		{"com.apple.dock", "minimize-to-application", []string{"-bool", "true"}},
		{"com.apple.screencapture", "type", []string{"-string", "png"}},
		{"com.apple.screencapture", "disable-shadow", []string{"-bool", "true"}},
	}
}

func Domains() []string {
	return []string{
		"NSGlobalDomain",
		"com.apple.dock",
		"com.apple.finder",
		"com.apple.screencapture",
		"com.apple.AppleMultitouchTrackpad",
		"com.apple.driver.AppleBluetoothMultitouch.trackpad",
		"com.mitchellh.ghostty",
		"com.googlecode.iterm2",
		"com.jordanbaird.Ice",
	}
}
