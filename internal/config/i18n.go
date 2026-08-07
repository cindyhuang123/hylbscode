package config

import (
	"errors"
	"sync/atomic"
)

// Supported UI languages.
const (
	LangEnglish = "en"
	LangChinese = "zh"
)

// Translation is a concrete set of UI strings for one language.
type Translation struct {
	PermissionTitle      string
	PermissionToolKey    string
	PermissionPathKey    string
	PermissionFileKey    string
	PermissionCommandKey string
	PermissionURLKey     string
	PermAllow            string
	PermAllowForSession  string
	PermDeny             string
	PermHelpSwitch       string
	PermHelpConfirm      string
	PermHelpAllow        string
	PermHelpAllowSession string
	PermHelpDeny         string
	PermHelpTab          string

	LanguageDialogTitle string
	LanguageHelpPrev    string
	LanguageHelpNext    string
	LanguageHelpSelect  string
	LanguageHelpClose   string

	// Chat list and message working states.
	TaskWorking         string
	Thinking            string
	WaitingToolResponse string
	BuildingToolCall    string
	Generating          string
	Loading             string

	// Help bar.
	Press        string
	ToSend       string
	ToAddNewLine string
	ToExitCancel string

	// Tool / action display names.
	ToolBash               string
	ToolEdit               string
	ToolFetch              string
	ToolGlob               string
	ToolGrep               string
	ToolList               string
	ToolSourcegraph        string
	ToolView               string
	ToolWrite              string
	ToolPatch              string
	ActionPreparingPrompt  string
	ActionBuildingCommand  string
	ActionPreparingEdit    string
	ActionWritingFetch     string
	ActionFindingFiles     string
	ActionSearchingContent string
	ActionListingDirectory string
	ActionSearchingCode    string
	ActionReadingFile      string
	ActionPreparingWrite   string
	ActionPreparingPatch   string
	ActionWorking          string

	// Sidebar.
	SessionLabel string

	LSPConfigLabel string
	ExecutingWord  string

	QuitQuestion    string
	QuitYes         string
	QuitNo          string
	QuitHelpSwitch  string
	QuitHelpConfirm string
	QuitHelpYes     string
	QuitHelpNo      string
	QuitHelpTab     string

	ContextLabel string
	CostLabel    string

	// GUI menu.
	GUIFileMenu       string
	GUISettingsItem   string
	GUIQuitItem       string
	GUIThemeMenu      string
	GUIThemeAuto      string
	GUIThemeLight     string
	GUIThemeDark      string
	GUIViewMenu       string
	GUIToggleLeftBar  string
	GUIToggleRightBar string
	GUILanguageMenu   string
	GUIEnglish        string
	GUIChinese        string
	GUICycleTheme     string
	GUISelectSession  string
	GUIAttachFile     string
	GUISwitchModel    string
	GUIContextCompact string

	// GUI right bar / status bar.
	GUITodoLabel    string
	GUITodoEmpty    string
	GUIVersionLabel string
	GUIModelLabel   string
	GUIWDLabel      string

	// GUI input / sidebar.
	GUIInputPlaceholder   string
	GUINewSession         string
	GUIDelete             string
	GUIDeleteSessionTitle string
	GUIDeleteSessionMsg   string
	GUIDone               string
	GUIDismiss            string

	// GUI help.
	GUIHelpMenu       string
	GUIHelpShortcuts  string
	GUISend           string
	GUINewline        string
	GUICancelResponse string
	GUIHistoryNav     string
	GUIScrollOutput   string

	// GUI provider setup.
	GUIProviderMenu    string
	GUIProviderTitle   string
	GUIProviderMissing string
	GUIProviderHint    string
	GUIProviderAPIKey  string
	GUIProviderBaseURL string
	GUIProviderSave    string
	GUIProviderNoKey   string
	GUIProviderSaved   string
	GUIProviderSelect  string
}

var (
	translations = map[string]Translation{
		LangEnglish: {
			PermissionTitle:      "Permission Required",
			PermissionToolKey:    "Tool",
			PermissionPathKey:    "Path",
			PermissionFileKey:    "File",
			PermissionCommandKey: "Command",
			PermissionURLKey:     "URL",
			PermAllow:            "Allow",
			PermAllowForSession:  "Allow for session",
			PermDeny:             "Deny",
			PermHelpSwitch:       "switch options",
			PermHelpConfirm:      "confirm",
			PermHelpAllow:        "allow",
			PermHelpAllowSession: "allow for session",
			PermHelpDeny:         "deny",
			PermHelpTab:          "switch options",

			LanguageDialogTitle: "Select Language",
			LanguageHelpPrev:    "previous language",
			LanguageHelpNext:    "next language",
			LanguageHelpSelect:  "select language",
			LanguageHelpClose:   "close",

			TaskWorking:         "Working...",
			Thinking:            "Thinking...",
			WaitingToolResponse: "Waiting for tool response...",
			BuildingToolCall:    "Building tool call...",
			Generating:          "Generating...",
			Loading:             "Loading...",

			Press:        "press ",
			ToSend:       " to send the message, ",
			ToAddNewLine: " and enter to add a new line",
			ToExitCancel: " to exit cancel",

			ToolBash:               "Bash",
			ToolEdit:               "Edit",
			ToolFetch:              "Fetch",
			ToolGlob:               "Glob",
			ToolGrep:               "Grep",
			ToolList:               "List",
			ToolSourcegraph:        "Sourcegraph",
			ToolView:               "View",
			ToolWrite:              "Write",
			ToolPatch:              "Patch",
			ActionPreparingPrompt:  "Preparing prompt...",
			ActionBuildingCommand:  "Building command...",
			ActionPreparingEdit:    "Preparing edit...",
			ActionWritingFetch:     "Writing fetch...",
			ActionFindingFiles:     "Finding files...",
			ActionSearchingContent: "Searching content...",
			ActionListingDirectory: "Listing directory...",
			ActionSearchingCode:    "Searching code...",
			ActionReadingFile:      "Reading file...",
			ActionPreparingWrite:   "Preparing write...",
			ActionPreparingPatch:   "Preparing patch...",
			ActionWorking:          "Working...",

			SessionLabel: "Session",

			LSPConfigLabel: "LSP Configuration",
			ExecutingWord:  "Executing",

			QuitQuestion:    "Are you sure you want to quit?",
			QuitYes:         "Yes",
			QuitNo:          "No",
			QuitHelpSwitch:  "switch options",
			QuitHelpConfirm: "confirm",
			QuitHelpYes:     "yes",
			QuitHelpNo:      "no",
			QuitHelpTab:     "switch options",

			ContextLabel: "Context",
			CostLabel:    "Cost",

			GUIFileMenu:       "File",
			GUISettingsItem:   "Settings",
			GUIQuitItem:       "Quit",
			GUIThemeMenu:      "Theme",
			GUIThemeAuto:      "Auto",
			GUIThemeLight:     "Light",
			GUIThemeDark:      "Dark",
			GUIViewMenu:       "View",
			GUIToggleLeftBar:  "Toggle Left Bar",
			GUIToggleRightBar: "Toggle Right Bar",
			GUILanguageMenu:   "Language",
			GUIEnglish:        "English",
			GUIChinese:        "中文",
			GUICycleTheme:     "Cycle Theme",
			GUISelectSession:  "Select Session",
			GUIAttachFile:     "Attach File",
			GUISwitchModel:    "Switch Model",
			GUIContextCompact: "Compact Context",

			GUITodoLabel:    "Todo",
			GUITodoEmpty:    "No todos for this session.",
			GUIVersionLabel: "Version",
			GUIModelLabel:   "Model",
			GUIWDLabel:      "WD",

			GUIInputPlaceholder:   "Type a message... Enter to send, Shift+Enter for newline",
			GUINewSession:         "New Session",
			GUIDelete:             "Delete",
			GUIDeleteSessionTitle: "Delete session",
			GUIDeleteSessionMsg:   "Delete \"%s\"? This cannot be undone.",
			GUIDone:               "Done",
			GUIDismiss:            "Cancel",

			GUIHelpMenu:       "Help",
			GUIHelpShortcuts:  "Keyboard Shortcuts",
			GUISend:           "Send",
			GUINewline:        "New line",
			GUICancelResponse: "Cancel response",
			GUIHistoryNav:     "Recall sent messages",
			GUIScrollOutput:   "Scroll output to top/bottom",

			GUIProviderMenu:    "Configure Provider",
			GUIProviderTitle:   "LLM Provider",
			GUIProviderMissing: "No LLM provider credentials detected.",
			GUIProviderHint:    "Configure an API key to get started, or close to continue without it.",
			GUIProviderSelect:  "Provider",
			GUIProviderAPIKey:  "API Key",
			GUIProviderBaseURL: "Base URL (optional)",
			GUIProviderSave:    "Save",
			GUIProviderNoKey:   "API key cannot be empty.",
			GUIProviderSaved:   "API key saved.",
		},
		LangChinese: {
			PermissionTitle:      "需要权限确认",
			PermissionToolKey:    "工具",
			PermissionPathKey:    "路径",
			PermissionFileKey:    "文件",
			PermissionCommandKey: "命令",
			PermissionURLKey:     "链接",
			PermAllow:            "允许",
			PermAllowForSession:  "允许本次会话",
			PermDeny:             "拒绝",
			PermHelpSwitch:       "切换选项",
			PermHelpConfirm:      "确认",
			PermHelpAllow:        "允许",
			PermHelpAllowSession: "允许本次会话",
			PermHelpDeny:         "拒绝",
			PermHelpTab:          "切换选项",

			LanguageDialogTitle: "选择语言",
			LanguageHelpPrev:    "上一个语言",
			LanguageHelpNext:    "下一个语言",
			LanguageHelpSelect:  "选择语言",
			LanguageHelpClose:   "关闭",

			TaskWorking:         "工作中...",
			Thinking:            "思考中...",
			WaitingToolResponse: "等待工具响应...",
			BuildingToolCall:    "正在构建工具调用...",
			Generating:          "生成中...",
			Loading:             "加载中...",

			Press:        "按 ",
			ToSend:       " 发送消息, ",
			ToAddNewLine: " 回车添加新行",
			ToExitCancel: " 退出取消",

			ToolBash:               "命令行",
			ToolEdit:               "编辑",
			ToolFetch:              "抓取",
			ToolGlob:               "查找文件",
			ToolGrep:               "搜索内容",
			ToolList:               "列出目录",
			ToolSourcegraph:        "代码搜索",
			ToolView:               "查看",
			ToolWrite:              "写入",
			ToolPatch:              "打补丁",
			ActionPreparingPrompt:  "准备提示词...",
			ActionBuildingCommand:  "构建命令...",
			ActionPreparingEdit:    "准备编辑...",
			ActionWritingFetch:     "获取中...",
			ActionFindingFiles:     "查找文件...",
			ActionSearchingContent: "搜索内容...",
			ActionListingDirectory: "列出目录...",
			ActionSearchingCode:    "搜索代码...",
			ActionReadingFile:      "读取文件...",
			ActionPreparingWrite:   "准备写入...",
			ActionPreparingPatch:   "准备打补丁...",
			ActionWorking:          "工作中...",

			SessionLabel: "会话",

			LSPConfigLabel: "LSP 配置",
			ExecutingWord:  "执行中",

			QuitQuestion:    "确定要退出吗?",
			QuitYes:         "是",
			QuitNo:          "否",
			QuitHelpSwitch:  "切换选项",
			QuitHelpConfirm: "确认",
			QuitHelpYes:     "是",
			QuitHelpNo:      "否",
			QuitHelpTab:     "切换选项",

			ContextLabel: "上下文",
			CostLabel:    "成本",

			GUIFileMenu:       "文件",
			GUISettingsItem:   "设置",
			GUIQuitItem:       "退出",
			GUIThemeMenu:      "主题",
			GUIThemeAuto:      "自动",
			GUIThemeLight:     "浅色",
			GUIThemeDark:      "深色",
			GUIViewMenu:       "视图",
			GUIToggleLeftBar:  "切换左侧栏",
			GUIToggleRightBar: "切换右侧栏",
			GUILanguageMenu:   "语言",
			GUIEnglish:        "English",
			GUIChinese:        "中文",
			GUICycleTheme:     "切换主题",
			GUISelectSession:  "选择会话",
			GUIAttachFile:     "添加附件",
			GUISwitchModel:    "切换模型",
			GUIContextCompact: "压缩上下文",

			GUITodoLabel:    "待办",
			GUITodoEmpty:    "此会话暂无待办。",
			GUIVersionLabel: "版本",
			GUIModelLabel:   "模型",
			GUIWDLabel:      "目录",

			GUIInputPlaceholder:   "输入消息... 回车发送, Shift+回车换行",
			GUINewSession:         "新会话",
			GUIDelete:             "删除",
			GUIDeleteSessionTitle: "删除会话",
			GUIDeleteSessionMsg:   "确定删除 \"%s\"? 此操作不可撤销。",
			GUIDone:               "完成",
			GUIDismiss:            "取消",

			GUIHelpMenu:       "帮助",
			GUIHelpShortcuts:  "键盘快捷键",
			GUISend:           "发送",
			GUINewline:        "换行",
			GUICancelResponse: "取消应答",
			GUIHistoryNav:     "翻看已发送消息",
			GUIScrollOutput:   "输出区滚动到顶部/底部",

			GUIProviderMenu:    "配置服务商",
			GUIProviderTitle:   "LLM 服务商",
			GUIProviderMissing: "未检测到 LLM 服务商凭据。",
			GUIProviderHint:    "配置 API Key 后开始使用，或关闭窗口继续。",
			GUIProviderSelect:  "服务商",
			GUIProviderAPIKey:  "API Key",
			GUIProviderBaseURL: "接口地址（可选）",
			GUIProviderSave:    "保存",
			GUIProviderNoKey:   "API Key 不能为空。",
			GUIProviderSaved:   "API Key 已保存。",
		},
	}

	currentLang atomic.Pointer[string]
)

// SetLanguage selects the active UI language. Unknown values fall back to
// English. This is safe to call concurrently.
func SetLanguage(lang string) error {
	if lang == "" {
		lang = LangEnglish
	}
	if _, ok := translations[lang]; !ok {
		return errors.New("unsupported language: " + lang)
	}
	currentLang.Store(&lang)
	return nil
}

// Tr returns the active translation set. It defaults to English.
func Tr() Translation {
	langPtr := currentLang.Load()
	if langPtr == nil {
		return translations[LangEnglish]
	}
	if t, ok := translations[*langPtr]; ok {
		return t
	}
	return translations[LangEnglish]
}

// CurrentLanguage returns the active language code ("en" or "zh").
func CurrentLanguage() string {
	langPtr := currentLang.Load()
	if langPtr == nil {
		return LangEnglish
	}
	return *langPtr
}

// AvailableLanguages returns the list of supported language codes.
func AvailableLanguages() []string {
	return []string{LangEnglish, LangChinese}
}

// UpdateLanguage switches the active UI language and persists it to the config
// file.
func UpdateLanguage(lang string) error {
	if err := SetLanguage(lang); err != nil {
		return err
	}
	if cfg == nil {
		return nil
	}
	cfg.TUI.Language = lang
	return updateCfgFile(func(c *Config) {
		c.TUI.Language = lang
	})
}
