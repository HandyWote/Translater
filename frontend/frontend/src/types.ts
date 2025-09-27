import {main} from '../wailsjs/go/models';

export type TranslationSource = 'manual' | 'screenshot';

export interface ScreenshotBounds {
	startX: number;
	startY: number;
	endX: number;
	endY: number;
	left: number;
	top: number;
	width: number;
	height: number;
}

export interface TranslationResult {
	originalText: string;
	translatedText: string;
	source: TranslationSource;
	timestamp: string;
	durationMs: number;
	bounds?: ScreenshotBounds;
}

export interface StatusMessage {
	stage: string;
	message: string;
}

export interface SettingsState {
	apiKeyOverride: string;
	apiBaseUrl: string;
	visionApiKeyOverride: string;
	visionApiBaseUrl: string;
	autoCopyResult: boolean;
	keepWindowOnTop: boolean;
	theme: string;
	showToastOnComplete: boolean;
	hotkeyCombination: string;
	extractPrompt: string;
	translatePrompt: string;
	translateModel: string;
	visionModel: string;
}

export const DEFAULT_API_BASE_URL = 'https://open.bigmodel.cn/api/paas/v4';
export const DEFAULT_TRANSLATE_MODEL = 'glm-4.5-flash';
export const DEFAULT_VISION_MODEL = 'glm-4v-flash';

export const DEFAULT_EXTRACT_PROMPT = `你是一个专业的视觉上下文分析专家，专门为高质量的翻译工作提供支持。你的核心任务是深度解读图片，为后续的精准翻译提供所有必要的上下文信息。

**你的分析必须包含以下两个层面：**

1.  **上下文背景分析：** 对图片进行极其详尽的描述，这将是决定翻译准确性的关键。请描述：
	   *   **场景与环境：** 图片描绘的是什么地方（如：熙熙攘攘的东京涩谷十字路口、一间安静的家庭书房、一个软件弹出窗口）？氛围如何（如：喜庆、严肃、科技感、温馨）？
	   *   **关键物体与布局：** 图片中有哪些主要和次要物体？（如：一张木桌上放着一台打开的笔记本电脑、一个冒着热气的马克杯、一本摊开的书）。描述它们的位置、颜色、材质和大致数量。
	   *   **视觉风格：** 图片是真实的照片、卡通插图、软件UI截图还是复古海报？色调是怎样的？
	   *   **潜在意图与受众：** 根据视觉元素，推断图片的可能目的（如：商业广告、教育材料、用户界面提示、个人备忘录）以及目标受众。
	   *   **任何其他可能影响文字含义的视觉线索。**

2.  **文字信息提取：** 精确无误地提取图片中的所有文字内容。
	   *   **绝对忠实：** 完全保留原始文字的格式，包括但不限于：换行符、空格、缩进、项目符号（•, -等）、标点符号和大小写。
	   *   **保持顺序：** 按照人类正常的阅读顺序（通常是从左到右，从上到下）提取和排列文字。如果有多栏或特殊布局，请清晰反映出来。
	   *   **不做任何修改：** 即使发现可能的拼写错误或语法问题，也请原样输出。

**最终，你必须将分析结果严格遵循以下JSON格式输出，不要有任何其他前言或后语：**

{
	 "background": "在这里提供你对图片背景的极其详尽的描述。",
	 "words": "在这里原封不动地输出提取的所有文字，保留所有格式。"
}

**请开始你的分析。**`;

export const DEFAULT_TRANSLATE_PROMPT = `你是一个专业的翻译AI，专门处理图像文字在特定背景下的翻译任务。你的核心能力是根据图片背景信息，对提取的文字进行语境化、贴切的中文翻译，同时严格保持原始格式。

## 数据处理流程
1. **分析背景**：仔细理解图片背景描述，把握场景、氛围、用途等关键信息
2. **语境化翻译**：根据背景信息选择最合适的翻译表达方式
3. **格式保持**：原样保留所有换行、空格、标点符号等格式元素
4. **专业适配**：针对不同场景（如菜单、标识、文档等）使用相应的专业术语

## 翻译原则
- **准确性**：在背景语境下选择最准确的中文表达
- **自然度**：翻译结果要符合中文表达习惯
- **一致性**：同一文档内术语保持统一
- **文化适配**：考虑中西方文化差异进行适当调整

!!!注意，只需要翻译words字段中的内容，background字段仅供参考，不要翻译background字段!!!
请始终按照这个标准处理用户提供的JSON数据。`;

export function defaultSettingsState(): SettingsState {
	return {
		apiKeyOverride: '',
		apiBaseUrl: DEFAULT_API_BASE_URL,
		visionApiKeyOverride: '',
		visionApiBaseUrl: DEFAULT_API_BASE_URL,
		autoCopyResult: true,
		keepWindowOnTop: false,
		theme: 'system',
		showToastOnComplete: true,
		hotkeyCombination: 'Alt+T',
		extractPrompt: DEFAULT_EXTRACT_PROMPT,
		translatePrompt: DEFAULT_TRANSLATE_PROMPT,
		translateModel: DEFAULT_TRANSLATE_MODEL,
		visionModel: DEFAULT_VISION_MODEL,
	};
}

export function mapTranslationResult(data: main.UITranslationResult | any): TranslationResult {
	const converted = data instanceof main.UITranslationResult ? data : main.UITranslationResult.createFrom(data);
	const timestamp = converted.timestamp instanceof Date ? converted.timestamp : new Date(converted.timestamp ?? Date.now());
	const rawBounds: any = (converted as any).bounds;
	const bounds = rawBounds
		? {
			startX: Number(rawBounds.startX) || 0,
			startY: Number(rawBounds.startY) || 0,
			endX: Number(rawBounds.endX) || 0,
			endY: Number(rawBounds.endY) || 0,
			left: Number(rawBounds.left) || 0,
			top: Number(rawBounds.top) || 0,
			width: Number(rawBounds.width) || 1,
			height: Number(rawBounds.height) || 1,
		}
		: undefined;
	return {
		originalText: converted.originalText ?? '',
		translatedText: converted.translatedText ?? '',
		source: (converted.source as TranslationSource) ?? 'manual',
		timestamp: timestamp.toISOString(),
		durationMs: Number.isFinite(converted.durationMs) ? converted.durationMs : 0,
		bounds,
	};
}

export function mapSettings(data: main.SettingsDTO | any): SettingsState {
	const converted = data instanceof main.SettingsDTO ? data : main.SettingsDTO.createFrom(data);
	const defaults = defaultSettingsState();
	return {
		apiKeyOverride: converted.apiKeyOverride ?? '',
		apiBaseUrl: converted.apiBaseUrl || defaults.apiBaseUrl,
		visionApiKeyOverride: converted.visionApiKeyOverride ?? '',
		visionApiBaseUrl: converted.visionApiBaseUrl || converted.apiBaseUrl || defaults.visionApiBaseUrl,
		autoCopyResult: Boolean(converted.autoCopyResult),
		keepWindowOnTop: Boolean(converted.keepWindowOnTop),
		theme: converted.theme || defaults.theme,
		showToastOnComplete: Boolean(converted.showToastOnComplete),
		hotkeyCombination: converted.hotkeyCombination || defaults.hotkeyCombination,
		extractPrompt: converted.extractPrompt || defaults.extractPrompt,
		translatePrompt: converted.translatePrompt || defaults.translatePrompt,
		translateModel: converted.translateModel || defaults.translateModel,
		visionModel: converted.visionModel || defaults.visionModel,
	};
}

export function toSettingsPayload(state: SettingsState): main.SettingsDTO {
	return main.SettingsDTO.createFrom({
		apiKeyOverride: state.apiKeyOverride,
		apiBaseUrl: state.apiBaseUrl,
		visionApiKeyOverride: state.visionApiKeyOverride,
		visionApiBaseUrl: state.visionApiBaseUrl,
		autoCopyResult: state.autoCopyResult,
		keepWindowOnTop: state.keepWindowOnTop,
		theme: state.theme,
		showToastOnComplete: state.showToastOnComplete,
		hotkeyCombination: state.hotkeyCombination,
		extractPrompt: state.extractPrompt,
		translatePrompt: state.translatePrompt,
		translateModel: state.translateModel,
		visionModel: state.visionModel,
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
