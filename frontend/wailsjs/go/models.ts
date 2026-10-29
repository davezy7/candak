export namespace models {
	
	export class AddDownloadRequest {
	    url: string;
	    filename: string;
	    saveDir: string;
	    connections: number;
	    headers?: Record<string, string>;
	    totalSize?: number;
	    resumable?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AddDownloadRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.filename = source["filename"];
	        this.saveDir = source["saveDir"];
	        this.connections = source["connections"];
	        this.headers = source["headers"];
	        this.totalSize = source["totalSize"];
	        this.resumable = source["resumable"];
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
	    headers?: Record<string, string>;
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
	        this.headers = source["headers"];
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
	export class GrabRequest {
	    url: string;
	    filename?: string;
	    headers?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new GrabRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.filename = source["filename"];
	        this.headers = source["headers"];
	    }
	}
	export class ProbeResult {
	    url: string;
	    filename: string;
	    totalSize: number;
	    resumable: boolean;
	    contentType: string;
	    headers?: Record<string, string>;
	    error?: string;
	
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
	        this.headers = source["headers"];
	        this.error = source["error"];
	    }
	}
	export class StartupConfig {
	    enabled: boolean;
	    minimized: boolean;
	
	    static createFrom(source: any = {}) {
	        return new StartupConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.minimized = source["minimized"];
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

export namespace updater {
	
	export class CheckResult {
	    currentVersion: string;
	    latestVersion: string;
	    updateAvailable: boolean;
	    publishedAt?: string;
	    releaseNotes?: string;
	    releaseUrl?: string;
	    assetName?: string;
	    assetUrl?: string;
	    assetSize?: number;
	
	    static createFrom(source: any = {}) {
	        return new CheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.updateAvailable = source["updateAvailable"];
	        this.publishedAt = source["publishedAt"];
	        this.releaseNotes = source["releaseNotes"];
	        this.releaseUrl = source["releaseUrl"];
	        this.assetName = source["assetName"];
	        this.assetUrl = source["assetUrl"];
	        this.assetSize = source["assetSize"];
	    }
	}

}

export namespace ytdl {
	
	export class FormatOption {
	    format_id: string;
	    resolution: string;
	    height: number;
	    ext: string;
	    filesize: number;
	    is_audio: boolean;
	    has_audio: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FormatOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.format_id = source["format_id"];
	        this.resolution = source["resolution"];
	        this.height = source["height"];
	        this.ext = source["ext"];
	        this.filesize = source["filesize"];
	        this.is_audio = source["is_audio"];
	        this.has_audio = source["has_audio"];
	    }
	}
	export class VideoMetadata {
	    id: string;
	    title: string;
	    thumbnail: string;
	    duration: number;
	    duration_str: string;
	    uploader: string;
	    options: FormatOption[];
	
	    static createFrom(source: any = {}) {
	        return new VideoMetadata(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.thumbnail = source["thumbnail"];
	        this.duration = source["duration"];
	        this.duration_str = source["duration_str"];
	        this.uploader = source["uploader"];
	        this.options = this.convertValues(source["options"], FormatOption);
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

