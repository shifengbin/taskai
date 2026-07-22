export namespace settings {
	
	export class TaskMenuItem {
	    id: string;
	    kind: string;
	    name: string;
	    command?: string;
	    arguments?: string[];
	    showTerminal: boolean;
	
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
	    }
	}
	export class Settings {
	    workspaceRoot: string;
	    taskTreeWidth: number;
	    colorScheme: string;
	    shellPath: string;
	    taskMenuItems: TaskMenuItem[];
	
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

