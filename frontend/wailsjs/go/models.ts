export namespace model {
	
	export class ExportResult {
	    output: string;
	    recordCount: number;
	    minTimeUsec: number;
	    maxTimeUsec: number;
	
	    static createFrom(source: any = {}) {
	        return new ExportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.output = source["output"];
	        this.recordCount = source["recordCount"];
	        this.minTimeUsec = source["minTimeUsec"];
	        this.maxTimeUsec = source["maxTimeUsec"];
	    }
	}
	export class Profile {
	    id: string;
	    browser: string;
	    name: string;
	    database: string;
	    engine: string;
	
	    static createFrom(source: any = {}) {
	        return new Profile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.browser = source["browser"];
	        this.name = source["name"];
	        this.database = source["database"];
	        this.engine = source["engine"];
	    }
	}

}

