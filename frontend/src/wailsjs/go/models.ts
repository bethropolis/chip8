export namespace settings {
	
	export class Settings {
	    clockSpeed: number;
	    displayColor: string;
	    scanlineEffect: boolean;
	    keyMap: Record<string, number>;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.clockSpeed = source["clockSpeed"];
	        this.displayColor = source["displayColor"];
	        this.scanlineEffect = source["scanlineEffect"];
	        this.keyMap = source["keyMap"];
	    }
	}

}

