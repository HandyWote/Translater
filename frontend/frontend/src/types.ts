import {main} from '../wailsjs/go/models';

export type TranslationSource = 'manual' | 'screenshot';

export interface TranslationResult {
	originalText: string;
	translatedText: string;
	source: TranslationSource;
	timestamp: string;
	durationMs: number;
}

export interface StatusMessage {
	stage: string;
	message: string;
}

export interface SettingsState {
	apiKeyOverride: string;
	targetLanguage: string;
	autoCopyResult: boolean;
	keepWindowOnTop: boolean;
	theme: string;
	showToastOnComplete: boolean;
	hotkeyCombination: string;
	extractPrompt: string;
	translatePrompt: string;
}

export function defaultSettingsState(): SettingsState {
	return {
		apiKeyOverride: '',
		targetLanguage: 'zh-CN',
		autoCopyResult: true,
		keepWindowOnTop: false,
		theme: 'system',
		showToastOnComplete: true,
		hotkeyCombination: 'Alt+T',
		extractPrompt: '请提取这张图片中的所有文字内容，只返回文字，不要添加任何其他说明。',
		translatePrompt: '请将以下文本翻译成中文，保持原文的格式和结构：',
	};
}

export function mapTranslationResult(data: main.UITranslationResult | any): TranslationResult {
	const converted = data instanceof main.UITranslationResult ? data : main.UITranslationResult.createFrom(data);
	const timestamp = converted.timestamp instanceof Date ? converted.timestamp : new Date(converted.timestamp ?? Date.now());
	return {
		originalText: converted.originalText ?? '',
		translatedText: converted.translatedText ?? '',
		source: (converted.source as TranslationSource) ?? 'manual',
		timestamp: timestamp.toISOString(),
		durationMs: Number.isFinite(converted.durationMs) ? converted.durationMs : 0,
	};
}

export function mapSettings(data: main.SettingsDTO | any): SettingsState {
	const converted = data instanceof main.SettingsDTO ? data : main.SettingsDTO.createFrom(data);
	const defaults = defaultSettingsState();
	return {
		apiKeyOverride: converted.apiKeyOverride ?? '',
		targetLanguage: converted.targetLanguage || defaults.targetLanguage,
		autoCopyResult: Boolean(converted.autoCopyResult),
		keepWindowOnTop: Boolean(converted.keepWindowOnTop),
		theme: converted.theme || defaults.theme,
		showToastOnComplete: Boolean(converted.showToastOnComplete),
		hotkeyCombination: converted.hotkeyCombination || defaults.hotkeyCombination,
		extractPrompt: converted.extractPrompt || defaults.extractPrompt,
		translatePrompt: converted.translatePrompt || defaults.translatePrompt,
	};
}

export function toSettingsPayload(state: SettingsState): main.SettingsDTO {
	return main.SettingsDTO.createFrom({
		apiKeyOverride: state.apiKeyOverride,
		targetLanguage: state.targetLanguage,
		autoCopyResult: state.autoCopyResult,
		keepWindowOnTop: state.keepWindowOnTop,
		theme: state.theme,
		showToastOnComplete: state.showToastOnComplete,
		hotkeyCombination: state.hotkeyCombination,
		extractPrompt: state.extractPrompt,
		translatePrompt: state.translatePrompt,
	});
}

export function formatTimestamp(iso: string): string {
	const date = new Date(iso);
	if (Number.isNaN(date.getTime())) {
		return iso;
	}
	return `${date.getMonth() + 1}/${date.getDate()} ${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}`;
}

export function formatDuration(durationMs: number): string {
	if (!Number.isFinite(durationMs) || durationMs <= 0) {
		return '';
	}
	if (durationMs < 1000) {
		return `${durationMs} ms`;
	}
	return `${(durationMs / 1000).toFixed(1)} s`;
}
