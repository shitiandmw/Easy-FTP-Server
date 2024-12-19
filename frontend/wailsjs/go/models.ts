export namespace main {
	
	export class DefaultConfig {
	    RootDir: string;
	    Username: string;
	    Password: string;
	    Port: string;
	    AutoStart: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DefaultConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootDir = source["RootDir"];
	        this.Username = source["Username"];
	        this.Password = source["Password"];
	        this.Port = source["Port"];
	        this.AutoStart = source["AutoStart"];
	    }
	}
	export class FTPConfig {
	    RootDir: string;
	    Username: string;
	    Password: string;
	    Port: string;
	    AutoStart: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FTPConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RootDir = source["RootDir"];
	        this.Username = source["Username"];
	        this.Password = source["Password"];
	        this.Port = source["Port"];
	        this.AutoStart = source["AutoStart"];
	    }
	}

}

