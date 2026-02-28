export namespace main {
	
	export class AuthSession {
	    nickname: string;
	    session_uuid: string;
	
	    static createFrom(source: any = {}) {
	        return new AuthSession(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nickname = source["nickname"];
	        this.session_uuid = source["session_uuid"];
	    }
	}
	export class Config {
	    newsFeedUrl: string;
	    apiBaseUrl: string;
	    server_host: string;
	    server_port: number;
	    sync_client_settings: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.newsFeedUrl = source["newsFeedUrl"];
	        this.apiBaseUrl = source["apiBaseUrl"];
	        this.server_host = source["server_host"];
	        this.server_port = source["server_port"];
	        this.sync_client_settings = source["sync_client_settings"];
	    }
	}
	export class NewsItem {
	    title: string;
	    link: string;
	    description: string;
	    published: string;
	
	    static createFrom(source: any = {}) {
	        return new NewsItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.link = source["link"];
	        this.description = source["description"];
	        this.published = source["published"];
	    }
	}

}

