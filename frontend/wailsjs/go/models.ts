export namespace application {
	
	export class TaskMenuCommandResult {
	    terminal?: terminal.Info;
	
	    static createFrom(source: any = {}) {
	        return new TaskMenuCommandResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.terminal = this.convertValues(source["terminal"], terminal.Info);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace fonts {
	
	export class Candidate {
	    family: string;
	    spacing: string;
	
	    static createFrom(source: any = {}) {
	        return new Candidate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.family = source["family"];
	        this.spacing = source["spacing"];
	    }
	}

}

export namespace quickinput {
	
	export class QuickInput {
	    id: string;
	    name: string;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new QuickInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.content = source["content"];
	    }
	}

}

export namespace repositorygit {

	export class Repository {
	    path: string;
	    branch?: string;
	    remote?: string;
	    notice?: string;
	    dirty: boolean;
	    hasUpstream: boolean;
	    remoteBranchExists: boolean;
	    synchronized: boolean;
	    action: string;

	    static createFrom(source: any = {}) {
	        return new Repository(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.branch = source["branch"];
	        this.remote = source["remote"];
	        this.notice = source["notice"];
	        this.dirty = source["dirty"];
	        this.hasUpstream = source["hasUpstream"];
	        this.remoteBranchExists = source["remoteBranchExists"];
	        this.synchronized = source["synchronized"];
	        this.action = source["action"];
	    }
	}

}

export namespace settings {
	
	export class LifecycleCommand {
	    id: string;
	    kind: string;
	    name: string;
	    command?: string;
	    arguments: string[];
	    chainArgumentMode: string;
	    documentation?: string;
	    applicableHooks: string[];
	
	    static createFrom(source: any = {}) {
	        return new LifecycleCommand(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.name = source["name"];
	        this.command = source["command"];
	        this.arguments = source["arguments"];
	        this.chainArgumentMode = source["chainArgumentMode"];
	        this.documentation = source["documentation"];
	        this.applicableHooks = source["applicableHooks"];
	    }
	}
	export class LifecycleCommandReference {
	    commandId: string;
	    arguments: string[];
	
	    static createFrom(source: any = {}) {
	        return new LifecycleCommandReference(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.commandId = source["commandId"];
	        this.arguments = source["arguments"];
	    }
	}
	export class LifecycleCommandChain {
	    id: string;
	    name: string;
	    commands: LifecycleCommandReference[];
	    commandIds?: string[];
	    applicableHooks: string[];
	
	    static createFrom(source: any = {}) {
	        return new LifecycleCommandChain(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.commands = this.convertValues(source["commands"], LifecycleCommandReference);
	        this.commandIds = source["commandIds"];
	        this.applicableHooks = source["applicableHooks"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class LifecyclePreset {
	    id: string;
	    name: string;
	    chains: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new LifecyclePreset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.chains = source["chains"];
	    }
	}
	export class TaskScript {
	    script?: string;
	    arguments?: string[];
	
	    static createFrom(source: any = {}) {
	        return new TaskScript(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.script = source["script"];
	        this.arguments = source["arguments"];
	    }
	}
	export class TaskMenuItem {
	    id: string;
	    kind: string;
	    name: string;
	    unshelveName?: string;
	    command?: string;
	    arguments?: string[];
	    showTerminal: boolean;
	    beforeScript?: TaskScript;
	    afterScript?: TaskScript;
	
	    static createFrom(source: any = {}) {
	        return new TaskMenuItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.name = source["name"];
	        this.unshelveName = source["unshelveName"];
	        this.command = source["command"];
	        this.arguments = source["arguments"];
	        this.showTerminal = source["showTerminal"];
	        this.beforeScript = this.convertValues(source["beforeScript"], TaskScript);
	        this.afterScript = this.convertValues(source["afterScript"], TaskScript);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TerminalNoteTemplate {
	    originalPrefix: string;
	    notePrefix: string;
	    listSuffix: string;

	    static createFrom(source: any = {}) {
	        return new TerminalNoteTemplate(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.originalPrefix = source["originalPrefix"];
	        this.notePrefix = source["notePrefix"];
	        this.listSuffix = source["listSuffix"];
	    }
	}
	export class TerminalShortcutStep {
	    kind: string;
	    text?: string;
	    key?: string;
	    modifiers?: string[];
	
	    static createFrom(source: any = {}) {
	        return new TerminalShortcutStep(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.text = source["text"];
	        this.key = source["key"];
	        this.modifiers = source["modifiers"];
	    }
	}
	export class TerminalShortcut {
	    id: string;
	    shortcut: string;
	    steps: TerminalShortcutStep[];
	    includePrograms?: string[];
	
	    static createFrom(source: any = {}) {
	        return new TerminalShortcut(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.shortcut = source["shortcut"];
	        this.steps = this.convertValues(source["steps"], TerminalShortcutStep);
	        this.includePrograms = source["includePrograms"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TerminalTheme {
	    background: string;
	    foreground: string;
	    cursor: string;
	    cursorAccent: string;
	    selectionBackground: string;
	    selectionForeground: string;
	    black: string;
	    red: string;
	    green: string;
	    yellow: string;
	    blue: string;
	    magenta: string;
	    cyan: string;
	    white: string;
	    brightBlack: string;
	    brightRed: string;
	    brightGreen: string;
	    brightYellow: string;
	    brightBlue: string;
	    brightMagenta: string;
	    brightCyan: string;
	    brightWhite: string;
	
	    static createFrom(source: any = {}) {
	        return new TerminalTheme(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.background = source["background"];
	        this.foreground = source["foreground"];
	        this.cursor = source["cursor"];
	        this.cursorAccent = source["cursorAccent"];
	        this.selectionBackground = source["selectionBackground"];
	        this.selectionForeground = source["selectionForeground"];
	        this.black = source["black"];
	        this.red = source["red"];
	        this.green = source["green"];
	        this.yellow = source["yellow"];
	        this.blue = source["blue"];
	        this.magenta = source["magenta"];
	        this.cyan = source["cyan"];
	        this.white = source["white"];
	        this.brightBlack = source["brightBlack"];
	        this.brightRed = source["brightRed"];
	        this.brightGreen = source["brightGreen"];
	        this.brightYellow = source["brightYellow"];
	        this.brightBlue = source["brightBlue"];
	        this.brightMagenta = source["brightMagenta"];
	        this.brightCyan = source["brightCyan"];
	        this.brightWhite = source["brightWhite"];
	    }
	}
	export class Settings {
	    workspaceRoot: string;
	    gitScanDepth: number;
	    taskTreeWidth: number;
	    colorScheme: string;
	    terminalFontFamily: string;
	    terminalFontSize: number;
	    terminalTheme: TerminalTheme;
	    terminalShortcuts: TerminalShortcut[];
	    terminalNoteTemplate: TerminalNoteTemplate;
	    windowMaximized: boolean;
	    shellPath: string;
	    taskMenuItems: TaskMenuItem[];
	    activeTaskStatus: string;
	    statusManagementMode: string;
	    statusManagementHTTPPort: number;
	    httpServiceEnabled: boolean;
	    lifecycleCommands: LifecycleCommand[];
	    lifecycleChains: LifecycleCommandChain[];
	    lifecyclePresets: LifecyclePreset[];
	    defaultLifecyclePresetId: string;
	    lifecycleDefaultChains?: Record<string, string>;
	    taskTemplates: task.TaskTemplate[];
	    activeTaskTemplateId: string;
	    presetVersion: number;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspaceRoot = source["workspaceRoot"];
	        this.gitScanDepth = source["gitScanDepth"];
	        this.taskTreeWidth = source["taskTreeWidth"];
	        this.colorScheme = source["colorScheme"];
	        this.terminalFontFamily = source["terminalFontFamily"];
	        this.terminalFontSize = source["terminalFontSize"];
	        this.terminalTheme = this.convertValues(source["terminalTheme"], TerminalTheme);
	        this.terminalShortcuts = this.convertValues(source["terminalShortcuts"], TerminalShortcut);
	        this.terminalNoteTemplate = this.convertValues(source["terminalNoteTemplate"], TerminalNoteTemplate);
	        this.windowMaximized = source["windowMaximized"];
	        this.shellPath = source["shellPath"];
	        this.taskMenuItems = this.convertValues(source["taskMenuItems"], TaskMenuItem);
	        this.activeTaskStatus = source["activeTaskStatus"];
	        this.statusManagementMode = source["statusManagementMode"];
	        this.statusManagementHTTPPort = source["statusManagementHTTPPort"];
	        this.httpServiceEnabled = source["httpServiceEnabled"];
	        this.lifecycleCommands = this.convertValues(source["lifecycleCommands"], LifecycleCommand);
	        this.lifecycleChains = this.convertValues(source["lifecycleChains"], LifecycleCommandChain);
	        this.lifecyclePresets = this.convertValues(source["lifecyclePresets"], LifecyclePreset);
	        this.defaultLifecyclePresetId = source["defaultLifecyclePresetId"];
	        this.lifecycleDefaultChains = source["lifecycleDefaultChains"];
	        this.taskTemplates = this.convertValues(source["taskTemplates"], task.TaskTemplate);
	        this.activeTaskTemplateId = source["activeTaskTemplateId"];
	        this.presetVersion = source["presetVersion"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	

}

export namespace task {
	
	export class ExtraInfoParameter {
	    key: string;
	    displayName: string;
	    required: boolean;
	    inputType: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new ExtraInfoParameter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.displayName = source["displayName"];
	        this.required = source["required"];
	        this.inputType = source["inputType"];
	        this.value = source["value"];
	    }
	}
	export class ExtraInfoField {
	    key: string;
	    displayName: string;
	    value?: string;
	    defaultValue?: string;
	
	    static createFrom(source: any = {}) {
	        return new ExtraInfoField(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.displayName = source["displayName"];
	        this.value = source["value"];
	        this.defaultValue = source["defaultValue"];
	    }
	}
	export class ExtraInfo {
	    id: string;
	    templateId: string;
	    catalogue: string;
	    fields: ExtraInfoField[];
	    parameters: ExtraInfoParameter[];
	
	    static createFrom(source: any = {}) {
	        return new ExtraInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.templateId = source["templateId"];
	        this.catalogue = source["catalogue"];
	        this.fields = this.convertValues(source["fields"], ExtraInfoField);
	        this.parameters = this.convertValues(source["parameters"], ExtraInfoParameter);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class ExtraInfoParameterDefinition {
	    key: string;
	    displayName: string;
	    required: boolean;
	    inputType: string;
	
	    static createFrom(source: any = {}) {
	        return new ExtraInfoParameterDefinition(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.displayName = source["displayName"];
	        this.required = source["required"];
	        this.inputType = source["inputType"];
	    }
	}
	export class ExtraInfoTemplate {
	    id: string;
	    catalogue: string;
	    fields: ExtraInfoField[];
	    parameters: ExtraInfoParameterDefinition[];
	    builtIn: boolean;
	    displayName?: string;
	    key?: string;
	    keyDisplayName?: string;
	    value?: string;
	
	    static createFrom(source: any = {}) {
	        return new ExtraInfoTemplate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.catalogue = source["catalogue"];
	        this.fields = this.convertValues(source["fields"], ExtraInfoField);
	        this.parameters = this.convertValues(source["parameters"], ExtraInfoParameterDefinition);
	        this.builtIn = source["builtIn"];
	        this.displayName = source["displayName"];
	        this.key = source["key"];
	        this.keyDisplayName = source["keyDisplayName"];
	        this.value = source["value"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class LifecycleExecution {
	    runId?: string;
	    revision?: number;
	    hook: string;
	    chainId: string;
	    currentCommandId?: string;
	    currentCommandName?: string;
	    currentIndex: number;
	    commandCount: number;
	    state: string;
	    error?: string;
	    workspaceRoot?: string;
	    workspacePath?: string;
	    workspaceOwnership?: string;
	    workspaceToken?: string;
	
	    static createFrom(source: any = {}) {
	        return new LifecycleExecution(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.runId = source["runId"];
	        this.revision = source["revision"];
	        this.hook = source["hook"];
	        this.chainId = source["chainId"];
	        this.currentCommandId = source["currentCommandId"];
	        this.currentCommandName = source["currentCommandName"];
	        this.currentIndex = source["currentIndex"];
	        this.commandCount = source["commandCount"];
	        this.state = source["state"];
	        this.error = source["error"];
	        this.workspaceRoot = source["workspaceRoot"];
	        this.workspacePath = source["workspacePath"];
	        this.workspaceOwnership = source["workspaceOwnership"];
	        this.workspaceToken = source["workspaceToken"];
	    }
	}
	export class TaskExtraInfo {
	    id: string;
	    informationId?: string;
	    templateId?: string;
	    catalogue: string;
	    displayName?: string;
	    fields: ExtraInfoField[];
	    parameters: ExtraInfoParameter[];
	    key?: string;
	    keyDisplayName?: string;
	    value?: string;
	
	    static createFrom(source: any = {}) {
	        return new TaskExtraInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.informationId = source["informationId"];
	        this.templateId = source["templateId"];
	        this.catalogue = source["catalogue"];
	        this.displayName = source["displayName"];
	        this.fields = this.convertValues(source["fields"], ExtraInfoField);
	        this.parameters = this.convertValues(source["parameters"], ExtraInfoParameter);
	        this.key = source["key"];
	        this.keyDisplayName = source["keyDisplayName"];
	        this.value = source["value"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Task {
	    id: string;
	    title: string;
	    description: string;
	    color: string;
	    status: string;
	    shelved: boolean;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    completedAt?: any;
	    workspaceRoot?: string;
	    workspacePath?: string;
	    extraInfo: TaskExtraInfo[];
	    taskTemplateId?: string;
	    templateFields: Record<string, any>;
	    lifecycleChains: Record<string, string>;
	    lifecycleExecution?: LifecycleExecution;
	
	    static createFrom(source: any = {}) {
	        return new Task(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.color = source["color"];
	        this.status = source["status"];
	        this.shelved = source["shelved"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.completedAt = this.convertValues(source["completedAt"], null);
	        this.workspaceRoot = source["workspaceRoot"];
	        this.workspacePath = source["workspacePath"];
	        this.extraInfo = this.convertValues(source["extraInfo"], TaskExtraInfo);
	        this.taskTemplateId = source["taskTemplateId"];
	        this.templateFields = source["templateFields"];
	        this.lifecycleChains = source["lifecycleChains"];
	        this.lifecycleExecution = this.convertValues(source["lifecycleExecution"], LifecycleExecution);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class TaskTemplateField {
	    key: string;
	    displayName: string;
	    inputType: string;
	    required: boolean;
	    defaultValue: any;
	    injectEnvironment: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TaskTemplateField(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.displayName = source["displayName"];
	        this.inputType = source["inputType"];
	        this.required = source["required"];
	        this.defaultValue = source["defaultValue"];
	        this.injectEnvironment = source["injectEnvironment"];
	    }
	}
	export class TaskTemplate {
	    id: string;
	    name: string;
	    fields: TaskTemplateField[];
	
	    static createFrom(source: any = {}) {
	        return new TaskTemplate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.fields = this.convertValues(source["fields"], TaskTemplateField);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace terminal {
	
	export class Info {
	    id: string;
	    taskId: string;
	    state: string;
	    command?: string;
	
	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.taskId = source["taskId"];
	        this.state = source["state"];
	        this.command = source["command"];
	    }
	}

}
