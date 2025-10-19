export namespace main {
	
	export class SettingsDTO {
	    apiKeyOverride: string;
	    autoCopyResult: boolean;
	    keepWindowOnTop: boolean;
	    theme: string;
	    showToastOnComplete: boolean;
	    enableStreamOutput: boolean;
	    hotkeyCombination: string;
	    extractPrompt: string;
	    translatePrompt: string;
	    apiBaseUrl: string;
	    translateModel: string;
	    visionModel: string;
	    visionApiBaseUrl: string;
	    visionApiKeyOverride: string;
	    useVisionForTranslation: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SettingsDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.apiKeyOverride = source["apiKeyOverride"];
	        this.autoCopyResult = source["autoCopyResult"];
	        this.keepWindowOnTop = source["keepWindowOnTop"];
	        this.theme = source["theme"];
	        this.showToastOnComplete = source["showToastOnComplete"];
	        this.enableStreamOutput = source["enableStreamOutput"];
	        this.hotkeyCombination = source["hotkeyCombination"];
	        this.extractPrompt = source["extractPrompt"];
	        this.translatePrompt = source["translatePrompt"];
	        this.apiBaseUrl = source["apiBaseUrl"];
	        this.translateModel = source["translateModel"];
	        this.visionModel = source["visionModel"];
	        this.visionApiBaseUrl = source["visionApiBaseUrl"];
	        this.visionApiKeyOverride = source["visionApiKeyOverride"];
	        this.useVisionForTranslation = source["useVisionForTranslation"];
	    }
	}
	export class UIScreenshotBounds {
	    startX: number;
	    startY: number;
	    endX: number;
	    endY: number;
	    left: number;
	    top: number;
	    width: number;
	    height: number;
	
	    static createFrom(source: any = {}) {
	        return new UIScreenshotBounds(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.startX = source["startX"];
	        this.startY = source["startY"];
	        this.endX = source["endX"];
	        this.endY = source["endY"];
	        this.left = source["left"];
	        this.top = source["top"];
	        this.width = source["width"];
	        this.height = source["height"];
	    }
	}
	export class UITranslationResult {
	    originalText: string;
	    translatedText: string;
	    source: string;
	    // Go type: time
	    timestamp: any;
	    durationMs: number;
	    bounds?: UIScreenshotBounds;
	
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
	        this.bounds = this.convertValues(source["bounds"], UIScreenshotBounds);
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

