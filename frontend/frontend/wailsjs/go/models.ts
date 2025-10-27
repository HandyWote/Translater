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
	    sourceLanguage: string;
	    targetLanguage: string;
	
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
	        this.sourceLanguage = source["sourceLanguage"];
	        this.targetLanguage = source["targetLanguage"];
	    }
	}

}

