package lib

import (
	"fmt"
	"golangutils"
	"golangutils/entity"
	"main/src/entities"
	"main/src/enums"
	"main/src/lib/platform"
	"main/src/lib/shared"
	"sort"

	"github.com/energye/systray"
)

var (
	menuJsonData     entities.MenuJson
	menu             []*systray.MenuItem
	isSystrayCreated = false
)

func refresh(forceLoadApps bool) {
	platform.InitPlatform(forceLoadApps)
	systray.ResetMenu()
	loadMenuJsonData()
	buildTrayApp()
	updateMenuJsonData()
}

func isValidItem(item entities.MenuItemJson) bool {
	if len(item.Name) < 1 {
		shared.ErrorNotify("Invalid Item: " + item.Name)
		return false
	}
	if item.Type != enums.WINDOWS_APPS && item.Type != enums.SHORTCUTS && item.Type != enums.COMMAND {
		shared.ErrorNotify("Invalid Item Type: " + item.Type + ", from name: " + item.Name)
		return false
	}
	return true
}

func buildMenuItem(items []entities.MenuItemJson, mainMenu *systray.MenuItem) {
	itemsInfo := []entities.ItemInfo{}
	for _, item := range items {
		if isValidItem(item) {
			appInfo, err := platform.GetItemInfo(item)
			if err == nil {
				itemsInfo = append(itemsInfo, appInfo)
			}
		}
	}
	if len(itemsInfo) == 0 {
		if mainMenu != nil {
			mainMenu.Disable()
		} else {
			buildEmptyMenu()
		}
	} else {
		for _, itemInfo := range itemsInfo {
			var menuItem *systray.MenuItem
			if mainMenu != nil {
				menuItem = mainMenu.AddSubMenuItem(itemInfo.Name, itemInfo.Name)
			} else {
				menuItem = systray.AddMenuItem(itemInfo.Name, itemInfo.Name)
			}
			if len(itemInfo.Icon) > 0 && golangutils.FileExist(itemInfo.Icon) {
				icon, err := golangutils.ReadFileInByte(itemInfo.Icon)
				if err != nil {
					shared.LoggerUtils.Error(err.Error())
				} else {
					menuItem.SetIcon(icon)
				}
			}
			menuItem.Click(func() {
				command := entity.Command{Cmd: itemInfo.Exec, Verbose: shared.EnableLogs, IsThrow: false}
				if shared.SystemUtils.IsWindows() {
					command.UsePowerShell = true
				} else if shared.SystemUtils.IsLinux() {
					command.UseBash = true
				}
				shared.ConsoleUtils.ExecRealTimeAsync(command)
			})
		}
	}
}

func buildSettingMenu() {
	settingsMenu := systray.AddMenuItem("Settings", "Settings")
	settingsMenu.AddSubMenuItem("Update Menu", "Update Menu for any changes").Click(func() {
		shared.InfoNotify("Processing...")
		refresh(true)
		shared.OkNotify("Processing, done.")
	})
	settingsMenu.AddSubMenuItem("Select/Change JSON file", "Select JSON configuration file").Click(func() {
		filename, err := shared.SelectFileDialog()
		if err != nil {
			shared.ErrorNotify(err.Error())
		} else {
			shared.InfoNotify("Processing...")
			golangutils.DeleteFile(shared.GetJsonFile())
			err := golangutils.CopyFile(filename, shared.GetJsonFile())
			if err != nil {
				shared.ErrorNotify(err.Error())
			} else {
				refresh(true)
			}
			shared.OkNotify("Processing, done.")
		}
	})
	enableLogsItem := settingsMenu.AddSubMenuItemCheckbox("Enable Logs", "Enable logs for most of operations", menuJsonData.EnableLogs)
	enableLogsItem.Click(func() {
		message := ""
		if enableLogsItem.Checked() {
			enableLogsItem.Uncheck()
			message = "disabled"
			shared.EnableLogs = false
		} else {
			enableLogsItem.Check()
			message = "enabled"
			shared.EnableLogs = true
		}
		message = fmt.Sprintf("All Logs was %s by user.", message)
		shared.LoggerUtils.Info(message)
		menuJsonData.EnableLogs = shared.EnableLogs
		updateMenuJsonData()
		shared.ShowMessageDialog(fmt.Sprintf("%s\n\nLog file is located in: <b>%s</b>", message, shared.GetLogFile()))
	})

	// About Settings
	aboutSettings := settingsMenu.AddSubMenuItem("About", "About")
	aboutSettings.AddSubMenuItem("Name: "+shared.ApplicationName, "").Disable()
	aboutSettings.AddSubMenuItem("Version: "+shared.ApplicationVersion, "").Disable()
	aboutSettings.AddSubMenuItem("Release Date: "+shared.ApplicationReleaseDate, "").Disable()
}

func buildEmptyMenu() {
	systray.AddMenuItem("Empty", "").Disable()
}

func buildOthersMenu() {
	if len(menuJsonData.Others) == 0 {
		buildEmptyMenu()
	} else {
		keys := make([]string, 0, len(menuJsonData.Others))
		for k := range menuJsonData.Others {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			mainMenu := systray.AddMenuItem(key, key)
			buildMenuItem(menuJsonData.Others[key], mainMenu)
		}
	}
}

func buildMenu() {
	buildOthersMenu()
	if len(menuJsonData.NoMenu) > 0 {
		systray.AddSeparator()
		buildMenuItem(menuJsonData.NoMenu, nil)
	}
	platform.ClearData()
	systray.AddSeparator()
	buildSettingMenu()
	systray.AddMenuItem("Exit", "Exit of the application").Click(func() {
		systray.Quit()
	})
}

func loadMenuJsonData() {
	menuJsonData = entities.MenuJson{}
	if golangutils.FileExist(shared.GetJsonFile()) {
		data, err := golangutils.ReadJsonFile[entities.MenuJson](shared.GetJsonFile())
		if err != nil {
			shared.ErrorNotify(err.Error())
		} else {
			menuJsonData = data
			menuJsonData.EnableLogs = shared.EnableLogs
			menuJsonData.NoMenu = shared.SortMenuItemByName(menuJsonData.NoMenu)
			for _, othersMenuItem := range menuJsonData.Others {
				othersMenuItem = shared.SortMenuItemByName(othersMenuItem)
			}
		}
	}
}

func updateMenuJsonData() {
	golangutils.WriteJsonFile(shared.GetJsonFile(), menuJsonData)
}

func buildTrayApp() {
	buildMenu()
	if !isSystrayCreated {
		fileByte, _ := golangutils.ReadFileInByte(shared.GetIcon())
		systray.SetIcon(fileByte)
		systray.SetTitle(shared.ApplicationName)
		systray.SetTooltip(shared.ApplicationName)
		systray.SetOnClick(func(menu systray.IMenu) {
			menu.ShowMenu()
		})
		systray.SetOnRClick(func(menu systray.IMenu) {
			menu.ShowMenu()
		})
		isSystrayCreated = true
		refresh(false)
	}
}

func Start() {
	platform.Validate()
	shared.LoadAppInformations()
	platform.InitPlatform(false)
	systray.Run(buildTrayApp, nil)
}
