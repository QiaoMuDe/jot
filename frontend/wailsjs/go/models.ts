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

export namespace models {
	
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
	    pinned: boolean;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	    deleted_at: gorm.DeletedAt;
	    tags?: Tag[];
	
	    static createFrom(source: any = {}) {
	        return new Note(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.content = source["content"];
	        this.pinned = source["pinned"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.deleted_at = this.convertValues(source["deleted_at"], gorm.DeletedAt);
	        this.tags = this.convertValues(source["tags"], Tag);
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
	
	export class DataStats {
	    total_notes: number;
	    trashed_notes: number;
	    pinned_notes: number;
	    total_tags: number;
	
	    static createFrom(source: any = {}) {
	        return new DataStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_notes = source["total_notes"];
	        this.trashed_notes = source["trashed_notes"];
	        this.pinned_notes = source["pinned_notes"];
	        this.total_tags = source["total_tags"];
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

}

