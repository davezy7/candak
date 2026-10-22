export namespace models {
	
	export class AddDownloadRequest {
	    url: string;
	    filename: string;
	    saveDir: string;
	    connections: number;
	
	    static createFrom(source: any = {}) {
	        return new AddDownloadRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.filename = source["filename"];
	        this.saveDir = source["saveDir"];
	        this.connections = source["connections"];
	    }
	}
	export class Chunk {
	    id: string;
	    downloadId: string;
	    chunkIndex: number;
	    startOffset: number;
	    endOffset: number;
	    currentOffset: number;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new Chunk(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.downloadId = source["downloadId"];
	        this.chunkIndex = source["chunkIndex"];
	        this.startOffset = source["startOffset"];
	        this.endOffset = source["endOffset"];
	        this.currentOffset = source["currentOffset"];
	        this.status = source["status"];
	    }
	}
	export class Download {
	    id: string;
	    url: string;
	    filename: string;
	    savePath: string;
	    totalSize: number;
	    downloadedSize: number;
	    status: string;
	    resumable: boolean;
	    totalChunks: number;
	    speed: number;
	    eta: number;
	    errorMessage: string;
	    createdAt: string;
	    updatedAt: string;
	    chunks?: Chunk[];
	
	    static createFrom(source: any = {}) {
	        return new Download(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.url = source["url"];
	        this.filename = source["filename"];
	        this.savePath = source["savePath"];
	        this.totalSize = source["totalSize"];
	        this.downloadedSize = source["downloadedSize"];
	        this.status = source["status"];
	        this.resumable = source["resumable"];
	        this.totalChunks = source["totalChunks"];
	        this.speed = source["speed"];
	        this.eta = source["eta"];
	        this.errorMessage = source["errorMessage"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.chunks = this.convertValues(source["chunks"], Chunk);
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
	export class ProbeResult {
	    url: string;
	    filename: string;
	    totalSize: number;
	    resumable: boolean;
	    contentType: string;
	
	    static createFrom(source: any = {}) {
	        return new ProbeResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.filename = source["filename"];
	        this.totalSize = source["totalSize"];
	        this.resumable = source["resumable"];
	        this.contentType = source["contentType"];
	    }
	}
	export class URLHistoryItem {
	    id: number;
	    url: string;
	    filename: string;
	    lastUsedAt: string;
	    downloadCount: number;
	
	    static createFrom(source: any = {}) {
	        return new URLHistoryItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.url = source["url"];
	        this.filename = source["filename"];
	        this.lastUsedAt = source["lastUsedAt"];
	        this.downloadCount = source["downloadCount"];
	    }
	}

}

