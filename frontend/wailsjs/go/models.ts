export namespace agent {
	
	export class ToolMeta {
	    Name: string;
	    Label: string;
	    Enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ToolMeta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.Label = source["Label"];
	        this.Enabled = source["Enabled"];
	    }
	}

}

export namespace gorm {
	
	export class DeletedAt {
	    // Go type: time
	    Time: any;
	    Valid: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DeletedAt(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Time = this.convertValues(source["Time"], null);
	        this.Valid = source["Valid"];
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

export namespace main {
	
	export class AIChatFileResult {
	    path: string;
	    name: string;
	    content: string;
	    size: number;
	    truncated: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new AIChatFileResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.content = source["content"];
	        this.size = source["size"];
	        this.truncated = source["truncated"];
	        this.error = source["error"];
	    }
	}
	export class CardRecallCheckResult {
	    ok: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new CardRecallCheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.message = source["message"];
	    }
	}
	export class FileImportResult {
	    path: string;
	    title: string;
	    note_id: number;
	    success: boolean;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new FileImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.title = source["title"];
	        this.note_id = source["note_id"];
	        this.success = source["success"];
	        this.error = source["error"];
	    }
	}
	export class SaveAIMessageResult {
	    msgID: number;
	    tokens: number;
	
	    static createFrom(source: any = {}) {
	        return new SaveAIMessageResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.msgID = source["msgID"];
	        this.tokens = source["tokens"];
	    }
	}
	export class TestMCPServerResult {
	    ok: boolean;
	    tool_num: number;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new TestMCPServerResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.tool_num = source["tool_num"];
	        this.message = source["message"];
	    }
	}
	export class VectorIndexOverview {
	    noteCount: number;
	    chunkCount: number;
	    sizeBytes: number;
	    totalNotes: number;
	    unindexedNotes: number;
	    staleNotes: number;
	    upToDateNotes: number;
	
	    static createFrom(source: any = {}) {
	        return new VectorIndexOverview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.noteCount = source["noteCount"];
	        this.chunkCount = source["chunkCount"];
	        this.sizeBytes = source["sizeBytes"];
	        this.totalNotes = source["totalNotes"];
	        this.unindexedNotes = source["unindexedNotes"];
	        this.staleNotes = source["staleNotes"];
	        this.upToDateNotes = source["upToDateNotes"];
	    }
	}
	export class VectorIndexStatus {
	    noteCount: number;
	    chunkCount: number;
	    sizeBytes: number;
	
	    static createFrom(source: any = {}) {
	        return new VectorIndexStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.noteCount = source["noteCount"];
	        this.chunkCount = source["chunkCount"];
	        this.sizeBytes = source["sizeBytes"];
	    }
	}

}

export namespace mcpserver {
	
	export class WarmupResult {
	    total: number;
	    warmed: number;
	    reused: number;
	    closed: number;
	    failed: number;
	    failed_msgs: string[];
	    tool_total: number;
	
	    static createFrom(source: any = {}) {
	        return new WarmupResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.warmed = source["warmed"];
	        this.reused = source["reused"];
	        this.closed = source["closed"];
	        this.failed = source["failed"];
	        this.failed_msgs = source["failed_msgs"];
	        this.tool_total = source["tool_total"];
	    }
	}

}

export namespace models {
	
	export class APIProfile {
	    id: number;
	    name: string;
	    base_url: string;
	    api_key: string;
	    is_active: boolean;
	    // Go type: time
	    created_at: any;
	
	    static createFrom(source: any = {}) {
	        return new APIProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.base_url = source["base_url"];
	        this.api_key = source["api_key"];
	        this.is_active = source["is_active"];
	        this.created_at = this.convertValues(source["created_at"], null);
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
	export class ImportMCPServerItem {
	    index: number;
	    name: string;
	    ok: boolean;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportMCPServerItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.name = source["name"];
	        this.ok = source["ok"];
	        this.error = source["error"];
	    }
	}
	export class MCPServer {
	    id: number;
	    name: string;
	    transport: string;
	    command: string;
	    args: string[];
	    env: Record<string, string>;
	    url: string;
	    headers: Record<string, string>;
	    enabled: boolean;
	    sort_order: number;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	
	    static createFrom(source: any = {}) {
	        return new MCPServer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.transport = source["transport"];
	        this.command = source["command"];
	        this.args = source["args"];
	        this.env = source["env"];
	        this.url = source["url"];
	        this.headers = source["headers"];
	        this.enabled = source["enabled"];
	        this.sort_order = source["sort_order"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
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
	export class Notebook {
	    id: number;
	    name: string;
	    sort_order: number;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	    deleted_at: gorm.DeletedAt;
	
	    static createFrom(source: any = {}) {
	        return new Notebook(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.sort_order = source["sort_order"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.deleted_at = this.convertValues(source["deleted_at"], gorm.DeletedAt);
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
	export class Tag {
	    id: number;
	    name: string;
	    color: string;
	    // Go type: time
	    created_at: any;
	    notes?: Note[];
	
	    static createFrom(source: any = {}) {
	        return new Tag(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.color = source["color"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.notes = this.convertValues(source["notes"], Note);
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
	export class Note {
	    id: number;
	    title: string;
	    content: string;
	    file_ext: string;
	    pinned: boolean;
	    notebook_id: number;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	    deleted_at: gorm.DeletedAt;
	    tags?: Tag[];
	    notebook?: Notebook;
	
	    static createFrom(source: any = {}) {
	        return new Note(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.content = source["content"];
	        this.file_ext = source["file_ext"];
	        this.pinned = source["pinned"];
	        this.notebook_id = source["notebook_id"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.deleted_at = this.convertValues(source["deleted_at"], gorm.DeletedAt);
	        this.tags = this.convertValues(source["tags"], Tag);
	        this.notebook = this.convertValues(source["notebook"], Notebook);
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
	
	export class ParseMCPServersResult {
	    ok: boolean;
	    error: string;
	    items: ImportMCPServerItem[];
	
	    static createFrom(source: any = {}) {
	        return new ParseMCPServersResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.error = source["error"];
	        this.items = this.convertValues(source["items"], ImportMCPServerItem);
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
	export class PasswordRecord {
	    id: number;
	    name: string;
	    username: string;
	    password: string;
	    url: string;
	    note: string;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	    deleted_at: gorm.DeletedAt;
	
	    static createFrom(source: any = {}) {
	        return new PasswordRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.url = source["url"];
	        this.note = source["note"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.deleted_at = this.convertValues(source["deleted_at"], gorm.DeletedAt);
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
	
	export class Todo {
	    id: number;
	    text: string;
	    done: boolean;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	
	    static createFrom(source: any = {}) {
	        return new Todo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.text = source["text"];
	        this.done = source["done"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
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

export namespace services {
	
	export class AIConfig {
	    base_url: string;
	    api_key: string;
	    model: string;
	
	    static createFrom(source: any = {}) {
	        return new AIConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.base_url = source["base_url"];
	        this.api_key = source["api_key"];
	        this.model = source["model"];
	    }
	}
	export class AISessionSummary {
	    id: number;
	    title: string;
	    context_tokens: number;
	    is_pinned: boolean;
	    last_message: string;
	    message_count: number;
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new AISessionSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.context_tokens = source["context_tokens"];
	        this.is_pinned = source["is_pinned"];
	        this.last_message = source["last_message"];
	        this.message_count = source["message_count"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class DataStats {
	    total_notes: number;
	    trashed_notes: number;
	    pinned_notes: number;
	    total_tags: number;
	    total_notebooks: number;
	    ai_sessions: number;
	    ai_messages: number;
	    total_tokens: number;
	    avg_response_time: number;
	    avg_thinking_time: number;
	    max_response_time: number;
	    db_size: number;
	    db_size_str: string;
	    total_todos: number;
	    completed_todos: number;
	    total_passwords: number;
	
	    static createFrom(source: any = {}) {
	        return new DataStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_notes = source["total_notes"];
	        this.trashed_notes = source["trashed_notes"];
	        this.pinned_notes = source["pinned_notes"];
	        this.total_tags = source["total_tags"];
	        this.total_notebooks = source["total_notebooks"];
	        this.ai_sessions = source["ai_sessions"];
	        this.ai_messages = source["ai_messages"];
	        this.total_tokens = source["total_tokens"];
	        this.avg_response_time = source["avg_response_time"];
	        this.avg_thinking_time = source["avg_thinking_time"];
	        this.max_response_time = source["max_response_time"];
	        this.db_size = source["db_size"];
	        this.db_size_str = source["db_size_str"];
	        this.total_todos = source["total_todos"];
	        this.completed_todos = source["completed_todos"];
	        this.total_passwords = source["total_passwords"];
	    }
	}
	export class GeneratedPassword {
	    password: string;
	    score: number;
	
	    static createFrom(source: any = {}) {
	        return new GeneratedPassword(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.password = source["password"];
	        this.score = source["score"];
	    }
	}
	export class ImportResult {
	    success_count: number;
	    fail_count: number;
	    skipped_count: number;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success_count = source["success_count"];
	        this.fail_count = source["fail_count"];
	        this.skipped_count = source["skipped_count"];
	        this.message = source["message"];
	    }
	}
	export class Message {
	    id: number;
	    role: string;
	    content: string;
	    reasoning_content: string;
	    thinking_elapsed: number;
	    total_elapsed: number;
	    tokens: number;
	    search_sources: string;
	    recall_cards: string;
	    tool_calls: string;
	    meta: string;
	
	    static createFrom(source: any = {}) {
	        return new Message(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.role = source["role"];
	        this.content = source["content"];
	        this.reasoning_content = source["reasoning_content"];
	        this.thinking_elapsed = source["thinking_elapsed"];
	        this.total_elapsed = source["total_elapsed"];
	        this.tokens = source["tokens"];
	        this.search_sources = source["search_sources"];
	        this.recall_cards = source["recall_cards"];
	        this.tool_calls = source["tool_calls"];
	        this.meta = source["meta"];
	    }
	}
	export class NoteRefInfo {
	    id: number;
	    title: string;
	    truncated: boolean;
	    notebook_name: string;
	
	    static createFrom(source: any = {}) {
	        return new NoteRefInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.truncated = source["truncated"];
	        this.notebook_name = source["notebook_name"];
	    }
	}
	export class NoteRefContext {
	    notes: NoteRefInfo[];
	    context: string;
	
	    static createFrom(source: any = {}) {
	        return new NoteRefContext(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.notes = this.convertValues(source["notes"], NoteRefInfo);
	        this.context = source["context"];
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
	
	export class PaginatedResult {
	    items: any;
	    total: number;
	    page: number;
	    page_size: number;
	
	    static createFrom(source: any = {}) {
	        return new PaginatedResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = source["items"];
	        this.total = source["total"];
	        this.page = source["page"];
	        this.page_size = source["page_size"];
	    }
	}
	export class PasswordGenOptions {
	    length: number;
	    count: number;
	    upper: boolean;
	    lower: boolean;
	    digits: boolean;
	    symbols: boolean;
	    excludeAmbiguous: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PasswordGenOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.length = source["length"];
	        this.count = source["count"];
	        this.upper = source["upper"];
	        this.lower = source["lower"];
	        this.digits = source["digits"];
	        this.symbols = source["symbols"];
	        this.excludeAmbiguous = source["excludeAmbiguous"];
	    }
	}
	export class PasswordStrengthResult {
	    score: number;
	    guesses: number;
	
	    static createFrom(source: any = {}) {
	        return new PasswordStrengthResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.score = source["score"];
	        this.guesses = source["guesses"];
	    }
	}
	export class SessionConfig {
	    model_name: string;
	    enable_thinking: boolean;
	    referenced_notes: string;
	    enabled_skills: string;
	    roleplay_notes: string;
	    recall_notebook_ids: string;
	
	    static createFrom(source: any = {}) {
	        return new SessionConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model_name = source["model_name"];
	        this.enable_thinking = source["enable_thinking"];
	        this.referenced_notes = source["referenced_notes"];
	        this.enabled_skills = source["enabled_skills"];
	        this.roleplay_notes = source["roleplay_notes"];
	        this.recall_notebook_ids = source["recall_notebook_ids"];
	    }
	}
	export class SettingsConfig {
	    theme: string;
	    font_family: string;
	    font_size: number;
	    code_highlight_theme: string;
	    note_open_fullscreen: boolean;
	    sort_order: string;
	    page_size: number;
	    cm_syntax_highlight: boolean;
	    ai_base_url: string;
	    ai_api_key: string;
	    ai_model: string;
	    ai_embed_base_url: string;
	    ai_embed_api_key: string;
	    ai_embed_model: string;
	    ai_thinking_enabled: boolean;
	    ai_card_recall_limit: number;
	    max_file_size: number;
	    ai_large_file_preview_threshold: number;
	    ai_agent_tools_disabled: string;
	    ai_agent_max_iterations: number;
	    trash_cleanup_retention_days: number;
	    log_level: number;
	    screen_lock_enabled: boolean;
	    screen_lock_password: string;
	    editor_word_wrap: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SettingsConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.font_family = source["font_family"];
	        this.font_size = source["font_size"];
	        this.code_highlight_theme = source["code_highlight_theme"];
	        this.note_open_fullscreen = source["note_open_fullscreen"];
	        this.sort_order = source["sort_order"];
	        this.page_size = source["page_size"];
	        this.cm_syntax_highlight = source["cm_syntax_highlight"];
	        this.ai_base_url = source["ai_base_url"];
	        this.ai_api_key = source["ai_api_key"];
	        this.ai_model = source["ai_model"];
	        this.ai_embed_base_url = source["ai_embed_base_url"];
	        this.ai_embed_api_key = source["ai_embed_api_key"];
	        this.ai_embed_model = source["ai_embed_model"];
	        this.ai_thinking_enabled = source["ai_thinking_enabled"];
	        this.ai_card_recall_limit = source["ai_card_recall_limit"];
	        this.max_file_size = source["max_file_size"];
	        this.ai_large_file_preview_threshold = source["ai_large_file_preview_threshold"];
	        this.ai_agent_tools_disabled = source["ai_agent_tools_disabled"];
	        this.ai_agent_max_iterations = source["ai_agent_max_iterations"];
	        this.trash_cleanup_retention_days = source["trash_cleanup_retention_days"];
	        this.log_level = source["log_level"];
	        this.screen_lock_enabled = source["screen_lock_enabled"];
	        this.screen_lock_password = source["screen_lock_password"];
	        this.editor_word_wrap = source["editor_word_wrap"];
	    }
	}

}

