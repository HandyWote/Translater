export namespace main {
	
	export class SettingsDTO {
	    apiKeyOverride: string;
	    targetLanguage: string;
	    autoCopyResult: boolean;
	    keepWindowOnTop: boolean;
	    theme: string;
	    showToastOnComplete: boolean;
	    hotkeyCombination: string;
	    extractPrompt: string;
	    translatePrompt: string;
	
	    static createFrom(source: any = {}) {
	        return new SettingsDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.apiKeyOverride = source["apiKeyOverride"];
	        this.targetLanguage = source["targetLanguage"];
	        this.autoCopyResult = source["autoCopyResult"];
	        this.keepWindowOnTop = source["keepWindowOnTop"];
	        this.theme = source["theme"];
	        this.showToastOnComplete = source["showToastOnComplete"];
	        this.hotkeyCombination = source["hotkeyCombination"];
	        this.extractPrompt = source["extractPrompt"];
	        this.translatePrompt = source["translatePrompt"];
	    }
	}
	export class UITranslationResult {
	    originalText: string;
	    translatedText: string;
	    source: string;
	    // Go type: time
	    timestamp: any;
	    durationMs: number;
	
	    static createFrom(source: any = {}) {
	        return new UITranslationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.originalText = source["originalText"];
	        this.translatedText = source["translatedText"];
	        this.source = source["source"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.durationMs = source["durationMs"];
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

