<script lang="ts" setup>
import {computed, onMounted, reactive, ref, watch} from 'vue';
import type {SettingsState} from '../types';
import {
	DEFAULT_API_BASE_URL,
	DEFAULT_EXTRACT_PROMPT,
	DEFAULT_TRANSLATE_MODEL,
	DEFAULT_TRANSLATE_PROMPT,
	DEFAULT_VISION_MODEL,
	defaultSettingsState,
} from '../types';

const props = defineProps<{
	settings: SettingsState;
	apiKeyMissing: boolean;
}>();

const emit = defineEmits<{
	(event: 'submit', value: SettingsState): void;
}>();

const form = reactive<SettingsState>({...props.settings});

const SECTION_STORAGE_KEY = 'settings-panel:sections';
const sectionDefaults = {
	api: true,
	models: false,
	behavior: true,
	prompts: false,
	hotkey: true,
	theme: true,
} as const;
type SectionKey = keyof typeof sectionDefaults;
const sections = reactive<Record<SectionKey, boolean>>({...sectionDefaults});

function loadSectionState() {
	if (typeof window === 'undefined') {
		return;
	}
	try {
		const raw = window.localStorage.getItem(SECTION_STORAGE_KEY);
		if (!raw) {
			return;
		}
		const stored = JSON.parse(raw);
		(Object.keys(sectionDefaults) as SectionKey[]).forEach((key) => {
			if (typeof stored?.[key] === 'boolean') {
				sections[key] = stored[key];
			}
		});
	} catch (error) {
		console.warn('恢复折叠状态失败', error);
	}
}

function persistSectionState(value: Record<SectionKey, boolean>) {
	if (typeof window === 'undefined') {
		return;
	}
	try {
		window.localStorage.setItem(SECTION_STORAGE_KEY, JSON.stringify(value));
	} catch (error) {
		console.warn('保存折叠状态失败', error);
	}
}

function isSectionExpanded(name: SectionKey): boolean {
	return sections[name] ?? true;
}

function toggleSection(name: SectionKey) {
	sections[name] = !isSectionExpanded(name);
}

onMounted(() => {
	loadSectionState();
});

watch(
	sections,
	(next) => {
		persistSectionState({...next});
	},
	{deep: true},
);

const DEFAULT_HOTKEY = 'Alt+T';
const DEFAULT_HOTKEY_MODIFIER = 'Alt';
const DEFAULT_HOTKEY_KEY = 'T';

const modifierOptions = [
	{label: '无修饰 (不推荐)', value: ''},
	{label: 'Alt', value: 'Alt'},
	{label: 'Ctrl', value: 'Ctrl'},
	{label: 'Shift', value: 'Shift'},
	{label: 'Ctrl + Alt', value: 'Ctrl+Alt'},
	{label: 'Ctrl + Shift', value: 'Ctrl+Shift'},
	{label: 'Alt + Shift', value: 'Alt+Shift'},
	{label: 'Ctrl + Alt + Shift', value: 'Ctrl+Alt+Shift'},
];

const modifierValues = new Set(modifierOptions.map((item) => item.value));

const functionKeyOptions = Array.from({length: 12}, (_, index) => {
	const value = `F${index + 1}`;
	return {label: value, value};
});

const keyOptions = [
	...'ABCDEFGHIJKLMNOPQRSTUVWXYZ'.split('').map((char) => ({label: char, value: char})),
	...functionKeyOptions,
];

const keyValues = new Set(keyOptions.map((item) => item.value));

const hotkeyModifiers = ref('');
const hotkeyKey = ref(DEFAULT_HOTKEY_KEY);

let syncingSelectors = false;
let syncingCombination = false;

function applyHotkeyToSelectors(combo: string) {
	const safeCombo = combo && combo.trim() ? combo : DEFAULT_HOTKEY;
	const parts = safeCombo
		.split('+')
		.map((part) => part.trim())
		.filter(Boolean);
	const key = parts.pop() ?? DEFAULT_HOTKEY_KEY;
	const modifier = parts.join('+');
	hotkeyModifiers.value = modifierValues.has(modifier) ? modifier : DEFAULT_HOTKEY_MODIFIER;
	hotkeyKey.value = keyValues.has(key) ? key : DEFAULT_HOTKEY_KEY;
}

watch(
	() => props.settings,
	(next) => {
		Object.assign(form, next);
	},
	{deep: true},
);

watch(
	() => form.hotkeyCombination,
	(combo) => {
		if (syncingCombination) {
			return;
		}
		syncingSelectors = true;
		applyHotkeyToSelectors(combo ?? '');
		syncingSelectors = false;
	},
	{immediate: true},
);

watch(
	[hotkeyModifiers, hotkeyKey],
	([modifier, key]) => {
		if (syncingSelectors) {
			return;
		}
		const segments: string[] = [];
		if (modifier) {
			segments.push(modifier);
		}
		if (key) {
			segments.push(key);
		}
		const nextValue = segments.join('+');
		if (nextValue === form.hotkeyCombination) {
			return;
		}
		syncingCombination = true;
		form.hotkeyCombination = nextValue;
		syncingCombination = false;
	},
);

const hotkeyPreview = computed(() => {
	const value = form.hotkeyCombination?.trim();
	return value ? value : DEFAULT_HOTKEY;
});

const extractPromptField = computed<string>({
	get() {
		return form.extractPrompt === DEFAULT_EXTRACT_PROMPT ? '' : form.extractPrompt;
	},
	set(value) {
		const trimmed = value?.trim() ?? '';
		form.extractPrompt = trimmed ? value : DEFAULT_EXTRACT_PROMPT;
	},
});

const translatePromptField = computed<string>({
	get() {
		return form.translatePrompt === DEFAULT_TRANSLATE_PROMPT ? '' : form.translatePrompt;
	},
	set(value) {
		const trimmed = value?.trim() ?? '';
		form.translatePrompt = trimmed ? value : DEFAULT_TRANSLATE_PROMPT;
	},
});

function handleSubmit(event: Event) {
	event.preventDefault();
	emit('submit', {...form});
}

function resetPrompts() {
	const defaults = defaultSettingsState();
	form.extractPrompt = defaults.extractPrompt;
	form.translatePrompt = defaults.translatePrompt;
}

function resetModels() {
	const defaults = defaultSettingsState();
	form.apiBaseUrl = defaults.apiBaseUrl;
	form.visionApiBaseUrl = defaults.visionApiBaseUrl;
	form.visionApiKeyOverride = defaults.visionApiKeyOverride;
	form.translateModel = defaults.translateModel;
	form.visionModel = defaults.visionModel;
}

const themeOptions = [
	{label: '跟随系统', value: 'system'},
	{label: '浅色', value: 'light'},
	{label: '深色', value: 'dark'},
];
</script>

<template>
	<form class="settings" @submit="handleSubmit">
	<section class="card" :class="{collapsed: !isSectionExpanded('api')}">
		<header>
			<div class="card-header-text">
				<h2>API 访问</h2>
				<p>{{ props.apiKeyMissing ? '未检测到 API Key，请输入有效凭证。' : '已配置 API Key，可直接使用截图和翻译功能。' }}</p>
			</div>
			<button type="button" class="collapse-toggle" @click="toggleSection('api')">
				{{ isSectionExpanded('api') ? '收起' : '展开' }}
			</button>
		</header>
		<div class="card-body" v-show="isSectionExpanded('api')">
			<div class="grid api-grid">
				<label class="field">
					<span>翻译 API Key</span>
					<input v-model="form.apiKeyOverride" type="password" placeholder="sk-xxxxxxxx" autocomplete="off"/>
					<small>保存后立即生效，仅保存在本机用户配置目录。</small>
				</label>
				<label class="field">
					<span>视觉 API Key（可选）</span>
					<input v-model="form.visionApiKeyOverride" type="password" placeholder="sk-xxxxxxxx" autocomplete="off"/>
					<small>留空时沿用左侧翻译 API Key。</small>
				</label>
			</div>
			<div class="grid api-grid">
				<label class="field">
					<span>翻译 API Base URL</span>
					<input v-model="form.apiBaseUrl" type="text" :placeholder="DEFAULT_API_BASE_URL" autocomplete="off"/>
					<small>输入兼容 OpenAI Chat Completions 的接口地址，结尾无需斜杠。</small>
				</label>
				<label class="field">
					<span>视觉 API Base URL</span>
					<input v-model="form.visionApiBaseUrl" type="text" :placeholder="DEFAULT_API_BASE_URL" autocomplete="off"/>
					<small>留空时沿用左侧接口地址，支持不同服务商。</small>
				</label>
			</div>
		</div>
	</section>

	<section class="card" :class="{collapsed: !isSectionExpanded('models')}">
		<header>
			<div class="card-header-text">
				<h2>模型与高级能力</h2>
				<p>管理翻译模型、视觉模型以及流式/视觉直译等特性。</p>
			</div>
			<button type="button" class="collapse-toggle" @click="toggleSection('models')">
				{{ isSectionExpanded('models') ? '收起' : '展开' }}
			</button>
		</header>
		<div class="card-body" v-show="isSectionExpanded('models')">
			<div class="grid model-grid">
				<label class="field">
					<span>翻译模型</span>
					<input v-model="form.translateModel" type="text" :placeholder="DEFAULT_TRANSLATE_MODEL" autocomplete="off"/>
					<small>用于文本翻译（Chat Completions）。</small>
				</label>
				<label class="field">
					<span>视觉模型</span>
					<input v-model="form.visionModel" type="text" :placeholder="DEFAULT_VISION_MODEL" autocomplete="off"/>
					<small>用于图像识别（多模态消息）。</small>
				</label>
			</div>
			<div class="advanced-toggles">
				<label class="toggle">
					<input v-model="form.enableStreamOutput" type="checkbox"/>
					<div>
						<strong>启用流式输出</strong>
						<span>实时推送翻译进度，适合长文本或逐句对照。</span>
					</div>
				</label>
				<label class="toggle">
					<input v-model="form.useVisionForTranslation" type="checkbox"/>
					<div>
						<strong>视觉模型直接翻译</strong>
						<span>跳过文本模型，由多模态模型直接输出译文。</span>
					</div>
				</label>
			</div>
			<div class="model-actions">
				<button type="button" class="ghost" @click="resetModels">恢复默认接口与模型</button>
			</div>
		</div>
	</section>

	<section class="card" :class="{collapsed: !isSectionExpanded('behavior')}">
		<header>
			<div class="card-header-text">
				<h2>翻译行为</h2>
				<p>控制复制、前置等自动化行为。</p>
			</div>
			<button type="button" class="collapse-toggle" @click="toggleSection('behavior')">
				{{ isSectionExpanded('behavior') ? '收起' : '展开' }}
			</button>
		</header>
		<div class="card-body" v-show="isSectionExpanded('behavior')">
			<div class="behavior-list">
				<label class="toggle">
					<input v-model="form.autoCopyResult" type="checkbox"/>
					<div>
						<strong>翻译结果自动复制</strong>
						<span>完成翻译后自动写入剪贴板。</span>
					</div>
				</label>
				<label class="toggle">
					<input v-model="form.keepWindowOnTop" type="checkbox"/>
					<div>
						<strong>窗口置顶显示</strong>
						<span>前台保持窗口，方便对照阅读。</span>
					</div>
				</label>
				<label class="toggle">
					<input v-model="form.showToastOnComplete" type="checkbox"/>
					<div>
						<strong>显示完成提示</strong>
						<span>翻译完成时弹出底部提醒。</span>
					</div>
				</label>
			</div>
		</div>
	</section>

	<section class="card" :class="{collapsed: !isSectionExpanded('prompts')}">
		<header>
			<div class="card-header-text">
				<h2>提示词管理</h2>
				<p>自定义文字识别与翻译阶段的提示词，适配不同语境。</p>
			</div>
			<button type="button" class="collapse-toggle" @click="toggleSection('prompts')">
				{{ isSectionExpanded('prompts') ? '收起' : '展开' }}
			</button>
		</header>
		<div class="card-body" v-show="isSectionExpanded('prompts')">
			<label class="field">
				<span>文字提取提示词</span>
				<textarea
					v-model="extractPromptField"
					class="prompt-textarea"
					rows="3"
					placeholder="这里输入文字提取作用的提示词（正常无需修改）"
				></textarea>
			</label>
			<label class="field">
				<span>翻译提示词</span>
				<textarea
					v-model="translatePromptField"
					class="prompt-textarea"
					rows="4"
					placeholder="这里输入翻译作用的提示词（正常无需修改）"
				></textarea>
			</label>
			<div class="prompt-actions">
				<button type="button" class="ghost" @click="resetPrompts">恢复默认提示词</button>
			</div>
		</div>
	</section>

	<section class="card" :class="{collapsed: !isSectionExpanded('hotkey')}">
		<header>
			<div class="card-header-text">
				<h2>全局热键</h2>
				<p>选择用于启动截图翻译的键位组合，保存后自动启用。</p>
			</div>
			<button type="button" class="collapse-toggle" @click="toggleSection('hotkey')">
				{{ isSectionExpanded('hotkey') ? '收起' : '展开' }}
			</button>
		</header>
		<div class="card-body" v-show="isSectionExpanded('hotkey')">
			<div class="grid hotkey-grid">
				<label class="field">
					<span>修饰键</span>
					<select v-model="hotkeyModifiers">
						<option v-for="option in modifierOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
					</select>
				</label>
				<label class="field">
					<span>主键</span>
					<select v-model="hotkeyKey">
						<option v-for="option in keyOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
					</select>
				</label>
			</div>
			<div class="hotkey-preview">
				<span class="hotkey-preview__label">当前组合</span>
				<strong class="hotkey-preview__value">{{ hotkeyPreview }}</strong>
				<small>确保与其他软件热键不冲突，保存设置后立即生效。</small>
			</div>
		</div>
	</section>

	<section class="card" :class="{collapsed: !isSectionExpanded('theme')}">
		<header>
			<div class="card-header-text">
				<h2>界面主题</h2>
				<p>根据喜好切换深浅色或跟随系统。</p>
			</div>
			<button type="button" class="collapse-toggle" @click="toggleSection('theme')">
				{{ isSectionExpanded('theme') ? '收起' : '展开' }}
			</button>
		</header>
		<div class="card-body" v-show="isSectionExpanded('theme')">
			<label class="field">
				<span>界面主题</span>
				<select v-model="form.theme">
					<option v-for="option in themeOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
				</select>
			</label>
		</div>
	</section>

		<div class="actions">
			<button type="submit" class="primary">保存设置</button>
		</div>
	</form>
</template>

<style scoped>
.settings {
	display: flex;
	flex-direction: column;
	gap: 1.3rem;
	color: var(--color-text-primary);
}

.card {
	background: var(--surface-elevated);
	border-radius: 18px;
	border: 1px solid var(--border-subtle);
	padding: 1.3rem 1.5rem;
	box-shadow: 0 16px 28px rgba(10, 10, 30, 0.22);
	display: flex;
	flex-direction: column;
	gap: 1rem;
}

.card header {
	display: flex;
	justify-content: space-between;
	align-items: flex-start;
	gap: 0.9rem;
}

.card-header-text {
	display: flex;
	flex-direction: column;
	gap: 0.35rem;
}

.card-body {
	display: flex;
	flex-direction: column;
	gap: 1rem;
}

header h2 {
	margin: 0;
	font-size: 1.1rem;
	font-weight: 600;
}

header p {
	margin: 0.35rem 0 0;
	font-size: 0.85rem;
	color: var(--color-text-tertiary);
}

.field {
	display: flex;
	flex-direction: column;
	gap: 0.45rem;
}

.field span {
	font-weight: 500;
	font-size: 0.92rem;
}

.collapse-toggle {
	border: none;
	background: transparent;
	color: var(--color-text-tertiary);
	font-size: 0.8rem;
	padding: 0.2rem 0.5rem;
	cursor: pointer;
	transition: color 0.18s ease;
}

.collapse-toggle:hover {
	color: var(--color-text-secondary);
}

input,
select,
textarea {
	border-radius: 12px;
	border: 1px solid var(--border-subtle);
	background: var(--surface-base);
	color: inherit;
	padding: 0.65rem 0.85rem;
	font-size: 0.95rem;
}

input:focus,
select:focus,
textarea:focus {
	outline: none;
	border-color: var(--accent);
	box-shadow: 0 0 0 2px rgba(20, 131, 255, 0.2);
}

small {
	font-size: 0.75rem;
	color: var(--color-text-tertiary);
}

.grid {
	display: grid;
	gap: 1rem;
	grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
}

.behavior-list {
	display: flex;
	flex-direction: column;
	gap: 1rem;
}

.advanced-toggles {
	display: flex;
	flex-direction: column;
	gap: 0.9rem;
}

.behavior-list .toggle,
.behavior-list .field {
	width: 100%;
}

.prompt-textarea {
	min-height: 120px;
	resize: vertical;
	line-height: 1.5;
}

.prompt-actions {
	display: flex;
	justify-content: flex-end;
	padding-top: 0.5rem;
}

.model-actions {
	display: flex;
	justify-content: flex-end;
	padding-top: 0.5rem;
}

.ghost {
	border: 1px solid var(--border-subtle);
	border-radius: 10px;
	background: transparent;
	color: var(--color-text-secondary);
	padding: 0.55rem 1.1rem;
	cursor: pointer;
	transition: background 0.18s ease;
}

.ghost:hover {
	background: var(--surface-hover);
}

.hotkey-grid {
	grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
}

.api-grid {
	grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
}

.model-grid {
	grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
}

.hotkey-preview {
	margin-top: 1rem;
	display: flex;
	flex-direction: column;
	gap: 0.4rem;
	padding: 0.85rem 1rem;
	border-radius: 14px;
	border: 1px dashed var(--border-subtle);
	background: var(--surface-hover);
}

.hotkey-preview__label {
	font-size: 0.78rem;
	color: var(--color-text-tertiary);
	letter-spacing: 0.08em;
	text-transform: uppercase;
}

.hotkey-preview__value {
	font-size: 1.2rem;
	font-weight: 600;
	letter-spacing: 0.1em;
}

.hotkey-preview small {
	font-size: 0.75rem;
	color: var(--color-text-tertiary);
}

.toggle {
	display: flex;
	gap: 0.75rem;
	padding: 0.85rem 1rem;
	border-radius: 14px;
	border: 1px solid var(--border-subtle);
	background: var(--surface-hover);
	align-items: center;
}

.toggle input {
	width: 20px;
	height: 20px;
	margin: 0;
}

.toggle strong {
	display: block;
	font-size: 0.95rem;
	margin-bottom: 0.15rem;
}

.toggle span {
	font-size: 0.8rem;
	color: var(--color-text-tertiary);
}

.actions {
	display: flex;
	justify-content: flex-end;
}

.primary {
	border: none;
	background: var(--accent);
	color: #fff;
	border-radius: 12px;
	padding: 0.75rem 1.8rem;
	font-size: 0.95rem;
	box-shadow: 0 14px 26px rgba(20, 131, 255, 0.24);
	cursor: pointer;
	transition: transform 0.18s ease;
}

.primary:hover {
	transform: translateY(-1px);
}

@media (max-width: 720px) {
	.grid {
		grid-template-columns: 1fr;
	}
}
</style>
