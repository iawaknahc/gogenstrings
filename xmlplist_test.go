package main

import (
	"reflect"
	"testing"
)

func TestInfoPlist(t *testing.T) {
	input := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleDevelopmentRegion</key>
	<string>$(DEVELOPMENT_LANGUAGE)</string>
	<key>CFBundleExecutable</key>
	<string>$(EXECUTABLE_NAME)</string>
	<key>CFBundleIdentifier</key>
	<string>$(PRODUCT_BUNDLE_IDENTIFIER)</string>
	<key>CFBundleInfoDictionaryVersion</key>
	<string>6.0</string>
	<key>CFBundleName</key>
	<string>$(PRODUCT_NAME)</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleShortVersionString</key>
	<string>1.0</string>
	<key>CFBundleVersion</key>
	<string>1</string>
	<key>LSRequiresIPhoneOS</key>
	<true/>
	<key>UILaunchStoryboardName</key>
	<string>LaunchScreen</string>
	<key>UIMainStoryboardFile</key>
	<string>Main</string>
	<key>UIRequiredDeviceCapabilities</key>
	<array>
		<string>armv7</string>
	</array>
	<key>UISupportedInterfaceOrientations</key>
	<array>
		<string>UIInterfaceOrientationPortrait</string>
		<string>UIInterfaceOrientationLandscapeLeft</string>
		<string>UIInterfaceOrientationLandscapeRight</string>
	</array>
	<key>UISupportedInterfaceOrientations~ipad</key>
	<array>
		<string>UIInterfaceOrientationPortrait</string>
		<string>UIInterfaceOrientationPortraitUpsideDown</string>
		<string>UIInterfaceOrientationLandscapeLeft</string>
		<string>UIInterfaceOrientationLandscapeRight</string>
	</array>
</dict>
</plist>
`
	actual, err := parseInfoPlist(input, "")
	expected := infoPlist{
		"CFBundleDevelopmentRegion":     "$(DEVELOPMENT_LANGUAGE)",
		"CFBundleExecutable":            "$(EXECUTABLE_NAME)",
		"CFBundleIdentifier":            "$(PRODUCT_BUNDLE_IDENTIFIER)",
		"CFBundleInfoDictionaryVersion": "6.0",
		"CFBundleName":                  "$(PRODUCT_NAME)",
		"CFBundlePackageType":           "APPL",
		"CFBundleShortVersionString":    "1.0",
		"CFBundleVersion":               "1",
		"UILaunchStoryboardName":        "LaunchScreen",
		"UIMainStoryboardFile":          "Main",
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Fail()
	}
}

func TestInfoPlistLocalizable(t *testing.T) {
	input := infoPlist{
		"CFBundleDevelopmentRegion":     "$(DEVELOPMENT_LANGUAGE)",
		"CFBundleExecutable":            "$(EXECUTABLE_NAME)",
		"CFBundleIdentifier":            "$(PRODUCT_BUNDLE_IDENTIFIER)",
		"CFBundleInfoDictionaryVersion": "6.0",
		"CFBundleName":                  "$(PRODUCT_NAME)",
		"CFBundleDisplayName":           "MyApp",
		"CFBundlePackageType":           "APPL",
		"CFBundleShortVersionString":    "1.0",
		"CFBundleVersion":               "1",
		"UILaunchStoryboardName":        "LaunchScreen",
		"UIMainStoryboardFile":          "Main",
		"NFCReaderUsageDescription":     "Use NFC",
	}
	actual := input.localizable()
	expected := infoPlist{
		"CFBundleDisplayName":       "MyApp",
		"NFCReaderUsageDescription": "Use NFC",
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fail()
	}
}

func TestInfoPlistToEntryMap(t *testing.T) {
	input := infoPlist{
		"CFBundleDisplayName":       "MyApp",
		"NFCReaderUsageDescription": "Use NFC",
	}
	actual := input.toEntryMap()
	expected := entryMap{
		"CFBundleDisplayName": entry{
			key:   "CFBundleDisplayName",
			value: "MyApp",
		},
		"NFCReaderUsageDescription": entry{
			key:   "NFCReaderUsageDescription",
			value: "Use NFC",
		},
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fail()
	}
}