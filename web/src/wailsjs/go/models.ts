export namespace desktop {
	
	export class CommandMeta {
	    id: string;
	    label: string;
	    category: string;
	    accelerator?: string;
	    paletteHidden?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CommandMeta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.category = source["category"];
	        this.accelerator = source["accelerator"];
	        this.paletteHidden = source["paletteHidden"];
	    }
	}
	export class DirtyPath {
	    path: string;
	    kind: string;
	
	    static createFrom(source: any = {}) {
	        return new DirtyPath(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.kind = source["kind"];
	    }
	}
	export class ExternalFileResult {
	    content: string;
	    insideLibrary: boolean;
	    relativePath: string;
	
	    static createFrom(source: any = {}) {
	        return new ExternalFileResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.content = source["content"];
	        this.insideLibrary = source["insideLibrary"];
	        this.relativePath = source["relativePath"];
	    }
	}
	export class FileFilter {
	    displayName: string;
	    pattern: string;
	
	    static createFrom(source: any = {}) {
	        return new FileFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.displayName = source["displayName"];
	        this.pattern = source["pattern"];
	    }
	}
	export class FileGitStatus {
	    dirty: boolean;
	    headAt: string;
	    hasHead: boolean;
	    untracked: boolean;
	    missing: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FileGitStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dirty = source["dirty"];
	        this.headAt = source["headAt"];
	        this.hasHead = source["hasHead"];
	        this.untracked = source["untracked"];
	        this.missing = source["missing"];
	    }
	}
	export class LibraryDirty {
	    plays: DirtyPath[];
	    sidecars: DirtyPath[];
	    other: DirtyPath[];
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new LibraryDirty(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.plays = this.convertValues(source["plays"], DirtyPath);
	        this.sidecars = this.convertValues(source["sidecars"], DirtyPath);
	        this.other = this.convertValues(source["other"], DirtyPath);
	        this.count = source["count"];
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
	export class LibraryNode {
	    path: string;
	    name: string;
	    kind: string;
	    children?: LibraryNode[];
	    updatedAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new LibraryNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.children = this.convertValues(source["children"], LibraryNode);
	        this.updatedAt = source["updatedAt"];
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
	export class PdfExportOptions {
	    pageSize: string;
	    style: string;
	    layout: string;
	    bookletGutter: string;
	
	    static createFrom(source: any = {}) {
	        return new PdfExportOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pageSize = source["pageSize"];
	        this.style = source["style"];
	        this.layout = source["layout"];
	        this.bookletGutter = source["bookletGutter"];
	    }
	}
	export class Preferences {
	    theme?: string;
	    previewHidden?: boolean;
	    spellcheckDisabled?: boolean;
	    sidebarCollapsed?: boolean;
	    sidebarWidth?: number;
	    lastDrawerTab?: string;
	    drawerDock?: string;
	    drawerRightWidth?: number;
	    exportPageSize?: string;
	    exportStyle?: string;
	    exportLayout?: string;
	    exportBookletGutter?: string;
	
	    static createFrom(source: any = {}) {
	        return new Preferences(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.previewHidden = source["previewHidden"];
	        this.spellcheckDisabled = source["spellcheckDisabled"];
	        this.sidebarCollapsed = source["sidebarCollapsed"];
	        this.sidebarWidth = source["sidebarWidth"];
	        this.lastDrawerTab = source["lastDrawerTab"];
	        this.drawerDock = source["drawerDock"];
	        this.drawerRightWidth = source["drawerRightWidth"];
	        this.exportPageSize = source["exportPageSize"];
	        this.exportStyle = source["exportStyle"];
	        this.exportLayout = source["exportLayout"];
	        this.exportBookletGutter = source["exportBookletGutter"];
	    }
	}
	export class Revision {
	    hash: string;
	    path: string;
	    message: string;
	    author: string;
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new Revision(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hash = source["hash"];
	        this.path = source["path"];
	        this.message = source["message"];
	        this.author = source["author"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class WindowState {
	    width?: number;
	    height?: number;
	
	    static createFrom(source: any = {}) {
	        return new WindowState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.width = source["width"];
	        this.height = source["height"];
	    }
	}
	export class lspWorkspaceEditJSON {
	    changes: Record<string, Array<lspTextEditJSON>>;
	
	    static createFrom(source: any = {}) {
	        return new lspWorkspaceEditJSON(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.changes = this.convertValues(source["changes"], Array<lspTextEditJSON>, true);
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
	export class lspCodeActionJSON {
	    title: string;
	    kind?: string;
	    isPreferred?: boolean;
	    edit?: lspWorkspaceEditJSON;
	
	    static createFrom(source: any = {}) {
	        return new lspCodeActionJSON(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.kind = source["kind"];
	        this.isPreferred = source["isPreferred"];
	        this.edit = this.convertValues(source["edit"], lspWorkspaceEditJSON);
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
	export class codeActionsResultJSON {
	    uri: string;
	    actions: lspCodeActionJSON[];
	
	    static createFrom(source: any = {}) {
	        return new codeActionsResultJSON(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uri = source["uri"];
	        this.actions = this.convertValues(source["actions"], lspCodeActionJSON);
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
	export class diagnosticJSON {
	    message: string;
	    severity: string;
	    line: number;
	    col: number;
	    endLine: number;
	    endCol: number;
	    code?: string;
	    quickFixes?: string[];
	
	    static createFrom(source: any = {}) {
	        return new diagnosticJSON(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.message = source["message"];
	        this.severity = source["severity"];
	        this.line = source["line"];
	        this.col = source["col"];
	        this.endLine = source["endLine"];
	        this.endCol = source["endCol"];
	        this.code = source["code"];
	        this.quickFixes = source["quickFixes"];
	    }
	}
	export class documentSymbolsResultJSON {
	    symbols: protocol.DocumentSymbol[];
	
	    static createFrom(source: any = {}) {
	        return new documentSymbolsResultJSON(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.symbols = this.convertValues(source["symbols"], protocol.DocumentSymbol);
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
	
	export class lspTextEditJSON {
	    range: protocol.Range;
	    newText: string;
	
	    static createFrom(source: any = {}) {
	        return new lspTextEditJSON(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.range = this.convertValues(source["range"], protocol.Range);
	        this.newText = source["newText"];
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
	
	export class parseErrorJSON {
	    message: string;
	    line: number;
	    col: number;
	    endLine: number;
	    endCol: number;
	
	    static createFrom(source: any = {}) {
	        return new parseErrorJSON(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.message = source["message"];
	        this.line = source["line"];
	        this.col = source["col"];
	        this.endLine = source["endLine"];
	        this.endCol = source["endCol"];
	    }
	}
	export class spellcheckContextJSON {
	    allowWords: string[];
	    ignoredRanges: protocol.Range[];
	
	    static createFrom(source: any = {}) {
	        return new spellcheckContextJSON(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.allowWords = source["allowWords"];
	        this.ignoredRanges = this.convertValues(source["ignoredRanges"], protocol.Range);
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
	export class upgradeResultJSON {
	    source: string;
	    changed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new upgradeResultJSON(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.changed = source["changed"];
	    }
	}

}

export namespace keys {
	
	export class Accelerator {
	    Key: string;
	    Modifiers: string[];
	
	    static createFrom(source: any = {}) {
	        return new Accelerator(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Key = source["Key"];
	        this.Modifiers = source["Modifiers"];
	    }
	}

}

export namespace menu {
	
	export class MenuItem {
	    Label: string;
	    Role: number;
	    Accelerator?: keys.Accelerator;
	    Type: string;
	    Disabled: boolean;
	    Hidden: boolean;
	    Checked: boolean;
	    SubMenu?: Menu;
	
	    static createFrom(source: any = {}) {
	        return new MenuItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Label = source["Label"];
	        this.Role = source["Role"];
	        this.Accelerator = this.convertValues(source["Accelerator"], keys.Accelerator);
	        this.Type = source["Type"];
	        this.Disabled = source["Disabled"];
	        this.Hidden = source["Hidden"];
	        this.Checked = source["Checked"];
	        this.SubMenu = this.convertValues(source["SubMenu"], Menu);
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
	export class Menu {
	    Items: MenuItem[];
	
	    static createFrom(source: any = {}) {
	        return new Menu(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Items = this.convertValues(source["Items"], MenuItem);
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

export namespace protocol {
	
	export class Command {
	    title: string;
	    command: string;
	    arguments?: any[];
	
	    static createFrom(source: any = {}) {
	        return new Command(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.command = source["command"];
	        this.arguments = source["arguments"];
	    }
	}
	export class Position {
	    line: number;
	    character: number;
	
	    static createFrom(source: any = {}) {
	        return new Position(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.line = source["line"];
	        this.character = source["character"];
	    }
	}
	export class Range {
	    start: Position;
	    end: Position;
	
	    static createFrom(source: any = {}) {
	        return new Range(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.start = this.convertValues(source["start"], Position);
	        this.end = this.convertValues(source["end"], Position);
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
	export class TextEdit {
	    range: Range;
	    newText: string;
	
	    static createFrom(source: any = {}) {
	        return new TextEdit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.range = this.convertValues(source["range"], Range);
	        this.newText = source["newText"];
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
	export class CompletionItem {
	    additionalTextEdits?: TextEdit[];
	    command?: Command;
	    commitCharacters?: string[];
	    tags?: number[];
	    data?: any;
	    deprecated?: boolean;
	    detail?: string;
	    documentation?: any;
	    filterText?: string;
	    insertText?: string;
	    insertTextFormat?: number;
	    insertTextMode?: number;
	    kind?: number;
	    label: string;
	    preselect?: boolean;
	    sortText?: string;
	    textEdit?: TextEdit;
	
	    static createFrom(source: any = {}) {
	        return new CompletionItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.additionalTextEdits = this.convertValues(source["additionalTextEdits"], TextEdit);
	        this.command = this.convertValues(source["command"], Command);
	        this.commitCharacters = source["commitCharacters"];
	        this.tags = source["tags"];
	        this.data = source["data"];
	        this.deprecated = source["deprecated"];
	        this.detail = source["detail"];
	        this.documentation = source["documentation"];
	        this.filterText = source["filterText"];
	        this.insertText = source["insertText"];
	        this.insertTextFormat = source["insertTextFormat"];
	        this.insertTextMode = source["insertTextMode"];
	        this.kind = source["kind"];
	        this.label = source["label"];
	        this.preselect = source["preselect"];
	        this.sortText = source["sortText"];
	        this.textEdit = this.convertValues(source["textEdit"], TextEdit);
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
	export class CompletionList {
	    isIncomplete: boolean;
	    items: CompletionItem[];
	
	    static createFrom(source: any = {}) {
	        return new CompletionList(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isIncomplete = source["isIncomplete"];
	        this.items = this.convertValues(source["items"], CompletionItem);
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
	export class DocumentSymbol {
	    name: string;
	    detail?: string;
	    kind: number;
	    tags?: number[];
	    deprecated?: boolean;
	    range: Range;
	    selectionRange: Range;
	    children?: DocumentSymbol[];
	
	    static createFrom(source: any = {}) {
	        return new DocumentSymbol(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.detail = source["detail"];
	        this.kind = source["kind"];
	        this.tags = source["tags"];
	        this.deprecated = source["deprecated"];
	        this.range = this.convertValues(source["range"], Range);
	        this.selectionRange = this.convertValues(source["selectionRange"], Range);
	        this.children = this.convertValues(source["children"], DocumentSymbol);
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

export namespace stats {
	
	export class CharacterStats {
	    name: string;
	    aliases?: string[];
	    lines: number;
	    dialogueWords: number;
	
	    static createFrom(source: any = {}) {
	        return new CharacterStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.aliases = source["aliases"];
	        this.lines = source["lines"];
	        this.dialogueWords = source["dialogueWords"];
	    }
	}
	export class RuntimeEstimate {
	    preset: string;
	    wordsPerMinute: number;
	    pauseFactor: number;
	    dialogueWords: number;
	    minutes: number;
	
	    static createFrom(source: any = {}) {
	        return new RuntimeEstimate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.preset = source["preset"];
	        this.wordsPerMinute = source["wordsPerMinute"];
	        this.pauseFactor = source["pauseFactor"];
	        this.dialogueWords = source["dialogueWords"];
	        this.minutes = source["minutes"];
	    }
	}
	export class Stats {
	    acts: number;
	    scenes: number;
	    songs: number;
	    totalWords: number;
	    dialogueWords: number;
	    lines: number;
	    stageDirections: number;
	    stageDirectionWords: number;
	    characters: CharacterStats[];
	    runtime: RuntimeEstimate;
	
	    static createFrom(source: any = {}) {
	        return new Stats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.acts = source["acts"];
	        this.scenes = source["scenes"];
	        this.songs = source["songs"];
	        this.totalWords = source["totalWords"];
	        this.dialogueWords = source["dialogueWords"];
	        this.lines = source["lines"];
	        this.stageDirections = source["stageDirections"];
	        this.stageDirectionWords = source["stageDirectionWords"];
	        this.characters = this.convertValues(source["characters"], CharacterStats);
	        this.runtime = this.convertValues(source["runtime"], RuntimeEstimate);
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

