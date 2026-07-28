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

export namespace settings {

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
	export class Settings {
	    workspaceRoot: string;
	    taskTreeWidth: number;
	    colorScheme: string;
	    shellPath: string;
	    taskMenuItems: TaskMenuItem[];
	    activeTaskStatus: string;
	    statusManagementMode: string;
	    statusManagementHTTPPort: number;
	    httpServiceEnabled: boolean;

	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspaceRoot = source["workspaceRoot"];
	        this.taskTreeWidth = source["taskTreeWidth"];
	        this.colorScheme = source["colorScheme"];
	        this.shellPath = source["shellPath"];
	        this.taskMenuItems = this.convertValues(source["taskMenuItems"], TaskMenuItem);
	        this.activeTaskStatus = source["activeTaskStatus"];
	        this.statusManagementMode = source["statusManagementMode"];
	        this.statusManagementHTTPPort = source["statusManagementHTTPPort"];
	        this.httpServiceEnabled = source["httpServiceEnabled"];
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
	    value: string;

	    static createFrom(source: any = {}) {
	        return new ExtraInfoParameter(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.displayName = source["displayName"];
	        this.required = source["required"];
	        this.value = source["value"];
	    }
	}
	export class ExtraInfoField {
	    key: string;
	    displayName: string;
	    value: string;

	    static createFrom(source: any = {}) {
	        return new ExtraInfoField(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.displayName = source["displayName"];
	        this.value = source["value"];
	    }
	}
	export class ExtraInfo {
	    id: string;
	    catalogue: string;
	    displayName: string;
	    fields: ExtraInfoField[];
	    parameters: ExtraInfoParameter[];
	    key?: string;
	    keyDisplayName?: string;
	    value?: string;

	    static createFrom(source: any = {}) {
	        return new ExtraInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
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


	export class ExtraInfoParameterDefinition {
	    key: string;
	    displayName: string;
	    required: boolean;

	    static createFrom(source: any = {}) {
	        return new ExtraInfoParameterDefinition(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.displayName = source["displayName"];
	        this.required = source["required"];
	    }
	}
	export class ExtraInfoTemplate {
	    id: string;
	    catalogue: string;
	    displayName: string;
	    fields: ExtraInfoField[];
	    parameters: ExtraInfoParameterDefinition[];
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
	        this.displayName = source["displayName"];
	        this.fields = this.convertValues(source["fields"], ExtraInfoField);
	        this.parameters = this.convertValues(source["parameters"], ExtraInfoParameterDefinition);
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
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    completedAt?: any;
	    workspaceRoot?: string;
	    workspacePath?: string;
	    extraInfo: ExtraInfo[];

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
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.completedAt = this.convertValues(source["completedAt"], null);
	        this.workspaceRoot = source["workspaceRoot"];
	        this.workspacePath = source["workspacePath"];
	        this.extraInfo = this.convertValues(source["extraInfo"], ExtraInfo);
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

	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.taskId = source["taskId"];
	        this.state = source["state"];
	    }
	}

}

