export namespace main {
	
	export class AuthSession {
	    nickname: string;
	    session_uuid: string;
	    access_token?: string;
	    profile_id?: string;
	    profile_name?: string;
	
	    static createFrom(source: any = {}) {
	        return new AuthSession(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nickname = source["nickname"];
	        this.session_uuid = source["session_uuid"];
	        this.access_token = source["access_token"];
	        this.profile_id = source["profile_id"];
	        this.profile_name = source["profile_name"];
	    }
	}
	export class Config {
	    apiBaseUrl: string;
	    server_host: string;
	    server_port: number;
	    socks_proxy_host: string;
	    socks_proxy_port: number;
	    sync_client_settings: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.apiBaseUrl = source["apiBaseUrl"];
	        this.server_host = source["server_host"];
	        this.server_port = source["server_port"];
	        this.socks_proxy_host = source["socks_proxy_host"];
	        this.socks_proxy_port = source["socks_proxy_port"];
	        this.sync_client_settings = source["sync_client_settings"];
	    }
	}
	export class LauncherUpdateInfo {
	    version: string;
	    mandatory: boolean;
	    changelog: string;
	    download_url: string;
	    size: number;
	    sha256: string;
	    current_version: string;
	
	    static createFrom(source: any = {}) {
	        return new LauncherUpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.mandatory = source["mandatory"];
	        this.changelog = source["changelog"];
	        this.download_url = source["download_url"];
	        this.size = source["size"];
	        this.sha256 = source["sha256"];
	        this.current_version = source["current_version"];
	    }
	}
	export class NewsItem {
	    text: string;
	    link?: string;
	    published?: string;
	
	    static createFrom(source: any = {}) {
	        return new NewsItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.link = source["link"];
	        this.published = source["published"];
	    }
	}
	export class NewsFeedResponse {
	    authenticated: boolean;
	    message?: string;
	    news?: NewsItem;
	    update?: any;
	
	    static createFrom(source: any = {}) {
	        return new NewsFeedResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.authenticated = source["authenticated"];
	        this.message = source["message"];
	        this.news = this.convertValues(source["news"], NewsItem);
	        this.update = source["update"];
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

