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

type CategoryKey = 'integration' | 'experience' | 'productivity' | 'appearance';

interface CategoryItem {
	key: CategoryKey;
	label: string;
	description: string;
	icon: string;
	sections: SectionKey[];
}

const categories: CategoryItem[] = [
	{key: 'integration', label: '服务能力', description: '统筹接口凭证与模型策略，确保端到端可用性。', icon: '🔌', sections: ['api', 'models']},
	{key: 'experience', label: '工作流体验', description: '调优翻译后的自动化动作与提示词，贴合团队流程。', icon: '⚙️', sections: ['behavior', 'prompts']},
	{key: 'productivity', label: '效率工具', description: '统一热键与交互方式，保持操作一致性。', icon: '⌨️', sections: ['hotkey']},
	{key: 'appearance', label: '界面主题', description: '设置主题与视觉偏好，营造舒适的使用体验。', icon: '🎨', sections: ['theme']},
];

const activeCategory = ref<CategoryKey>('integration');
const currentCategory = computed<CategoryItem>(() => categories.find((item) => item.key === activeCategory.value) ?? categories[0]);
const visibleSections = computed<SectionKey[]>(() => currentCategory.value.sections);
const validationError = ref<string | null>(null);

function activateCategory(name: CategoryKey) {
	activeCategory.value = name;
	const target = categories.find((item) => item.key === name);
	if (!target) {
		return;
	}
	const hasExpanded = target.sections.some((key) => isSectionExpanded(key));
	if (!hasExpanded && target.sections.length > 0) {
		sections[target.sections[0]] = true;
	}
}

function isCategoryActive(name: CategoryKey): boolean {
	return activeCategory.value === name;
}

function isSectionVisible(name: SectionKey): boolean {
	return visibleSections.value.includes(name);
}

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
	{label: '无修饰（不推荐）', value: ''},
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

const showTranslateModelField = computed(() => !form.useVisionForTranslation);
const showTranslateApiFields = computed(() => !form.useVisionForTranslation);
const hasVisionKey = computed(() => Boolean(form.visionApiKeyOverride?.trim() || form.apiKeyOverride?.trim()));

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

const enableCustomPrompts = ref(
	form.extractPrompt !== DEFAULT_EXTRACT_PROMPT || form.translatePrompt !== DEFAULT_TRANSLATE_PROMPT,
);

let syncingPrompts = false;

watch(enableCustomPrompts, (enabled) => {
	if (enabled) {
		return;
	}
	syncingPrompts = true;
	form.extractPrompt = DEFAULT_EXTRACT_PROMPT;
	form.translatePrompt = DEFAULT_TRANSLATE_PROMPT;
	syncingPrompts = false;
});

watch(
	[() => form.extractPrompt, () => form.translatePrompt],
	([extract, translate]) => {
		if (syncingPrompts) {
			return;
		}
		enableCustomPrompts.value = !(extract === DEFAULT_EXTRACT_PROMPT && translate === DEFAULT_TRANSLATE_PROMPT);
	},
);

watch(
	[
		() => form.useVisionForTranslation,
		() => form.translateModel,
		() => form.visionModel,
		() => form.visionApiKeyOverride,
		() => form.apiKeyOverride,
	],
	() => {
		validationError.value = null;
	},
);

function handleSubmit(event: Event) {
	event.preventDefault();
	validationError.value = null;
	const translateModel = form.translateModel?.trim();
	const visionModel = form.visionModel?.trim();
	if (form.useVisionForTranslation) {
		if (!visionModel) {
			validationError.value = '视觉直出模式下需设置视觉模型名称，或关闭该模式。';
			return;
		}
		if (!hasVisionKey.value) {
			validationError.value = '请配置视觉专用或通用 API Key，以便正常调用视觉模型。';
			return;
		}
	} else if (!translateModel) {
		validationError.value = '请填写翻译模型名称，或启用视觉直出模式。';
		return;
	}
	emit('submit', {...form});
}

function resetPrompts() {
	const defaults = defaultSettingsState();
	form.extractPrompt = defaults.extractPrompt;
	form.translatePrompt = defaults.translatePrompt;
	enableCustomPrompts.value = false;
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
		<div class="settings-layout">
			
			<aside class="settings-nav">
				<div class="settings-toolbar__info">
					<h1>设置中心</h1>
				</div>
				<div class="settings-nav__header">
					<strong>配置分组</strong>
				</div>
				<nav class="settings-nav__list">
					<button
						v-for="category in categories"
						:key="category.key"
						type="button"
						class="settings-nav__item"
						:class="{active: isCategoryActive(category.key)}"
						@click="activateCategory(category.key)"
					>
						<span class="settings-nav__icon" aria-hidden="true">{{ category.icon }}</span>
						<div class="settings-nav__text">
							<span class="settings-nav__label">{{ category.label }}</span>
							<span class="settings-nav__desc">{{ category.description }}</span>
						</div>
					</button>
				</nav>
				<div class="settings-toolbar__actions">
					<span v-if="validationError" class="settings-toolbar__alert">{{ validationError }}</span>
					<button type="submit" class="primary">保存设置</button>
				</div>
			</aside>
			<div class="settings-content">
				<header class="settings-content__intro">
					<h2>{{ currentCategory.label }}</h2>
					<p>{{ currentCategory.description }}</p>
				</header>

				<section
					v-if="isSectionVisible('api')"
					class="card"
					:class="{collapsed: !isSectionExpanded('api')}"
				>
					<header>
						<div class="card-header-text">
							<h2>接口与凭证</h2>
							<p>{{ props.apiKeyMissing ? '未检测到 API Key，请录入可用凭证以启用服务。' : '凭证已配置，可直接使用截图与翻译能力。' }}</p>
						</div>
						<button type="button" class="collapse-toggle" @click="toggleSection('api')">
							{{ isSectionExpanded('api') ? '收起' : '展开' }}
						</button>
					</header>
					<div class="card-body" v-show="isSectionExpanded('api')">
						<div class="grid api-grid">
							<label class="field">
								<span>视觉 API Key（可选）</span>
								<input v-model="form.visionApiKeyOverride" type="password" placeholder="sk-xxxxxxxx" autocomplete="off"/>
								<small>默认用于视觉直出调用，如启用文本模型可再单独配置翻译 Key。</small>
							</label>
							<label v-if="showTranslateApiFields" class="field">
								<span>翻译 API Key</span>
								<input v-model="form.apiKeyOverride" type="password" placeholder="sk-xxxxxxxx" autocomplete="off"/>
								<small>保存后即时生效，仅存储于当前用户配置目录。</small>
							</label>
						</div>
						<div class="grid api-grid">
							<label class="field">
								<span>视觉 API Base URL</span>
								<input v-model="form.visionApiBaseUrl" type="text" :placeholder="DEFAULT_API_BASE_URL" autocomplete="off"/>
								<small>支持为视觉模型独立配置地址，留空时沿用翻译接口地址。</small>
							</label>
							<label v-if="showTranslateApiFields" class="field">
								<span>翻译 API Base URL</span>
								<input v-model="form.apiBaseUrl" type="text" :placeholder="DEFAULT_API_BASE_URL" autocomplete="off"/>
								<small>请输入兼容 OpenAI Chat Completions 的接口地址，结尾无需斜杠。</small>
							</label>
						</div>
					</div>
				</section>

				<section
					v-if="isSectionVisible('models')"
					class="card"
					:class="{collapsed: !isSectionExpanded('models')}"
				>
					<header>
						<div class="card-header-text">
							<h2>模型能力</h2>
						</div>
						<button type="button" class="collapse-toggle" @click="toggleSection('models')">
							{{ isSectionExpanded('models') ? '收起' : '展开' }}
						</button>
					</header>
					<div class="card-body" v-show="isSectionExpanded('models')">
						<div v-if="showTranslateModelField" class="grid model-grid">
							<label class="field">
								<span>视觉模型</span>
								<input v-model="form.visionModel" type="text" :placeholder="DEFAULT_VISION_MODEL" autocomplete="off"/>
								<small>用于多模态识别与翻译直出。</small>
							</label>
							<label class="field">
								<span>翻译模型</span>
								<input v-model="form.translateModel" type="text" :placeholder="DEFAULT_TRANSLATE_MODEL" autocomplete="off"/>
								<small>用于文本翻译（Chat Completions）。</small>
							</label>
						</div>
						<div v-else class="grid model-grid">
							<label class="field">
								<span>视觉模型</span>
								<input v-model="form.visionModel" type="text" :placeholder="DEFAULT_VISION_MODEL" autocomplete="off"/>
								<small>用于多模态识别与翻译直出。</small>
							</label>
						</div>
						<label class="toggle toggle--primary">
							<input v-model="form.useVisionForTranslation" type="checkbox"/>
							<div>
								<strong>视觉直出</strong>
							</div>
						</label>
						<label class="toggle">
							<input v-model="form.enableStreamOutput" type="checkbox"/>
							<div>
								<strong>流式输出</strong>
							</div>
						</label>
						<div class="model-actions">
							<button type="button" class="ghost" @click="resetModels">恢复默认接口与模型</button>
						</div>
					</div>
				</section>

				<section
					v-if="isSectionVisible('behavior')"
					class="card"
					:class="{collapsed: !isSectionExpanded('behavior')}"
				>
					<header>
						<div class="card-header-text">
							<h2>工作流行为</h2>
							<p>控制翻译完成后的动作逻辑，保持团队协作节奏。</p>
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
									<strong>自动复制译文</strong>
									<span>翻译完成后写入剪贴板，便于直接粘贴到工作文档。</span>
								</div>
							</label>
							<label class="toggle">
								<input v-model="form.keepWindowOnTop" type="checkbox"/>
								<div>
									<strong>窗口保持置顶</strong>
									<span>让结果面板始终可见，提升校对效率。</span>
								</div>
							</label>
							<label class="toggle">
								<input v-model="form.showToastOnComplete" type="checkbox"/>
								<div>
									<strong>显示完成提示</strong>
									<span>屏幕底部弹出完成提示，及时知晓处理状态。</span>
								</div>
							</label>
						</div>
					</div>
				</section>

				<section
					v-if="isSectionVisible('prompts')"
					class="card"
					:class="{collapsed: !isSectionExpanded('prompts')}"
				>
					<header>
						<div class="card-header-text">
							<h2>提示词管理</h2>
							<p>针对视觉直出与文本兜底流程，定制提示词让上下文更贴合业务术语。</p>
						</div>
						<button type="button" class="collapse-toggle" @click="toggleSection('prompts')">
							{{ isSectionExpanded('prompts') ? '收起' : '展开' }}
						</button>
					</header>
					<div class="card-body" v-show="isSectionExpanded('prompts')">
						<label class="toggle">
							<input v-model="enableCustomPrompts" type="checkbox"/>
							<div>
								<strong>启用自定义提示词</strong>
								<span>覆盖默认策略，适配专有术语与行业语境。</span>
							</div>
						</label>
						<div v-if="enableCustomPrompts" class="prompt-fields">
							<label class="field">
								<span>视觉理解提示词</span>
								<textarea
									v-model="extractPromptField"
									class="prompt-textarea"
									rows="3"
									placeholder="输入用于视觉理解 / OCR 阶段的提示词"
								></textarea>
							</label>
							<label v-if="showTranslateModelField" class="field">
								<span>文本翻译提示词</span>
								<textarea
									v-model="translatePromptField"
									class="prompt-textarea"
									rows="4"
									placeholder="输入用于文本翻译阶段的提示词"
								></textarea>
							</label>
							<div v-else class="callout callout--info">
								<p>当前采用视觉直出策略，无需配置额外的文本翻译提示词。</p>
							</div>
							<div class="prompt-actions">
								<button type="button" class="ghost" @click="resetPrompts">恢复默认提示词</button>
							</div>
						</div>
						<div v-else class="callout callout--info">
							<p>将沿用官方默认提示词，已针对视觉直出和文本兜底双流程调优。</p>
						</div>
					</div>
				</section>

				<section
					v-if="isSectionVisible('hotkey')"
					class="card"
					:class="{collapsed: !isSectionExpanded('hotkey')}"
				>
					<header>
						<div class="card-header-text">
							<h2>全局热键</h2>
							<p>统一按键方案，团队成员无需重新记忆操作流程。</p>
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
							<small>请避免与常用软件冲突，保存后即时应用系统热键。</small>
						</div>
					</div>
				</section>

				<section
					v-if="isSectionVisible('theme')"
					class="card"
					:class="{collapsed: !isSectionExpanded('theme')}"
				>
					<header>
						<div class="card-header-text">
							<h2>界面主题</h2>
							<p>为团队选择统一视觉风格，或跟随系统自动切换。</p>
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
			</div>
		</div>
	</form>
</template>


<style scoped>
.settings {
	display: flex;
	flex-direction: column;
	gap: 1.5rem;
	padding: 0 2rem 2rem;
	color: var(--color-text-primary);
}

.settings-toolbar {
	position: sticky;
	top: 0;
	z-index: 4;
	display: flex;
	align-items: flex-start;
	justify-content: space-between;
	gap: 1.5rem;
	padding: 1.2rem 0 1rem;
	background: var(--surface-base);
	border-bottom: 1px solid var(--border-subtle);
}

.settings-toolbar__info {
	display: flex;
	flex-direction: column;
	gap: 0.4rem;
}

.settings-toolbar__info h1 {
	margin: 0;
	font-size: 1.32rem;
	font-weight: 600;
	letter-spacing: 0.01em;
}

.settings-toolbar__info p {
	margin: 0;
	font-size: 0.88rem;
	color: var(--color-text-secondary);
	line-height: 1.5;
}

.settings-toolbar__actions {
	display: flex;
	align-items: center;
	justify-content: flex-end;
	gap: 0.85rem;
	flex-wrap: wrap;
}

.settings-toolbar__alert {
	display: inline-flex;
	align-items: center;
	gap: 0.4rem;
	padding: 0.45rem 0.8rem;
	border-radius: 999px;
	background: rgba(255, 86, 48, 0.12);
	border: 1px solid rgba(255, 86, 48, 0.28);
	color: #d03a16;
	font-size: 0.82rem;
	line-height: 1.2;
}

.settings-layout {
	display: grid;
	grid-template-columns: 240px minmax(0, 1fr);
	gap: 1.75rem;
	align-items: flex-start;
	padding-top: 1.25rem;
}

.settings-nav {
	position: sticky;
	top: 92px;
	display: flex;
	flex-direction: column;
	gap: 1.25rem;
	background: var(--surface-elevated);
	border-radius: 18px;
	border: 1px solid var(--border-strong);
	padding: 1.5rem 1.25rem;
	box-shadow: 0 14px 28px rgba(8, 12, 32, 0.14);
}

.settings-nav__header {
	display: flex;
	flex-direction: column;
	gap: 0.35rem;
}

.settings-nav__header strong {
	font-size: 0.94rem;
	font-weight: 600;
}

.settings-nav__header p {
	margin: 0;
	font-size: 0.78rem;
	color: var(--color-text-tertiary);
	line-height: 1.5;
}

.settings-nav__list {
	display: flex;
	flex-direction: column;
	gap: 0.6rem;
}

.settings-nav__item {
	display: flex;
	align-items: flex-start;
	gap: 0.85rem;
	width: 100%;
	text-align: left;
	padding: 0.75rem 0.95rem;
	border-radius: 14px;
	border: 1px solid var(--border-subtle);
	background: transparent;
	color: inherit;
	cursor: pointer;
	transition: border-color 0.2s ease, background 0.2s ease, color 0.2s ease, box-shadow 0.2s ease;
}

.settings-nav__item:hover {
	border-color: var(--border-strong);
	background: var(--surface-hover);
}

.settings-nav__item.active {
	border-color: var(--accent);
	background: rgba(20, 131, 255, 0.12);
	color: var(--accent-strong);
	box-shadow: 0 10px 22px rgba(20, 131, 255, 0.18);
}

.settings-nav__icon {
	font-size: 1.2rem;
	line-height: 1;
}

.settings-nav__text {
	display: flex;
	flex-direction: column;
	gap: 0.18rem;
}

.settings-nav__label {
	font-size: 0.92rem;
	font-weight: 600;
}

.settings-nav__desc {
	font-size: 0.78rem;
	color: var(--color-text-tertiary);
	line-height: 1.35;
}

.settings-content {
	display: flex;
	flex-direction: column;
	gap: 1.35rem;
}

.settings-content__intro {
	display: flex;
	flex-direction: column;
	gap: 0.3rem;
	margin-bottom: 0.3rem;
}

.settings-content__intro h2 {
	margin: 0;
	font-size: 1.32rem;
	font-weight: 600;
}

.settings-content__intro p {
	margin: 0;
	font-size: 0.9rem;
	color: var(--color-text-secondary);
	max-width: 560px;
	line-height: 1.6;
}

.card {
	background: var(--surface-elevated);
	border-radius: 18px;
	border: 1px solid var(--border-strong);
	padding: 1.35rem 1.55rem;
	box-shadow: 0 14px 26px rgba(10, 14, 40, 0.16);
	display: flex;
	flex-direction: column;
	gap: 1.2rem;
}

.card.collapsed {
	box-shadow: 0 8px 18px rgba(10, 14, 40, 0.12);
}

.card header {
	display: flex;
	justify-content: space-between;
	align-items: flex-start;
	gap: 0.9rem;
	padding-bottom: 0.65rem;
	border-bottom: 1px solid var(--border-subtle);
}

.card-header-text {
	display: flex;
	flex-direction: column;
	gap: 0.25rem;
}

.card header h2 {
	margin: 0;
	font-size: 1rem;
	font-weight: 600;
	letter-spacing: 0.01em;
}

.card header p {
	margin: 0;
	font-size: 0.82rem;
	color: var(--color-text-tertiary);
	line-height: 1.5;
}

.card-body {
	display: flex;
	flex-direction: column;
	gap: 1rem;
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
	font-size: 0.78rem;
	padding: 0.25rem 0.5rem;
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

.prompt-fields {
	display: flex;
	flex-direction: column;
	gap: 1rem;
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
	padding-top: 0.35rem;
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

.callout {
	padding: 0.85rem 1rem;
	border-radius: 14px;
	border: 1px solid var(--border-subtle);
	background: var(--surface-hover);
	color: var(--color-text-secondary);
	font-size: 0.82rem;
	line-height: 1.5;
}

.callout--warning {
	border-color: rgba(255, 186, 0, 0.5);
	background: rgba(255, 186, 0, 0.08);
	color: #a36200;
}

.callout--info {
	border-color: rgba(20, 131, 255, 0.18);
	background: rgba(20, 131, 255, 0.08);
	color: var(--color-text-secondary);
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
	transition: border-color 0.2s ease, background 0.2s ease;
}

.toggle--primary {
	border-color: rgba(20, 131, 255, 0.32);
	background: rgba(20, 131, 255, 0.08);
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

.toggle-followup {
	margin-top: -0.4rem;
	margin-left: 2.2rem;
	margin-right: 0.2rem;
	padding: 0.65rem 0.95rem;
	border-left: 2px solid var(--border-strong);
	background: rgba(16, 22, 40, 0.04);
	border-radius: 10px;
	color: var(--color-text-secondary);
	font-size: 0.78rem;
	line-height: 1.45;
}

.primary {
	align-items: flex-start;
	gap: 0.85rem;
	width: 100%;
	text-align: center;
	padding: 0.75rem 0.95rem;
	border-radius: 14px;
	border: 1px solid var(--border-subtle);
	background: var(--accent);
	color: #fff;
	font-size: 0.95rem;
	box-shadow: 0 16px 30px rgba(20, 131, 255, 0.28);
	cursor: pointer;
	transition: transform 0.18s ease, box-shadow 0.18s ease;
}

.primary:hover {
	transform: translateY(-1px);
	box-shadow: 0 18px 36px rgba(20, 131, 255, 0.32);
}

@media (max-width: 1080px) {
	.settings {
		padding: 0 1.5rem 1.8rem;
	}

	.settings-layout {
		grid-template-columns: 1fr;
	}

	.settings-nav {
		position: relative;
		top: auto;
	}
}

@media (max-width: 720px) {
	.settings {
		padding: 0 1.2rem 1.4rem;
	}

	.settings-toolbar {
		flex-direction: column;
		align-items: flex-start;
		gap: 0.9rem;
	}

	.settings-toolbar__actions {
		width: 100%;
		justify-content: space-between;
	}

	.settings-nav {
		padding: 1.2rem;
	}

	.grid {
		grid-template-columns: 1fr;
	}
}
</style>
