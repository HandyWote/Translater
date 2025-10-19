<script lang="ts" setup>
import {computed, onBeforeUnmount, onMounted, ref} from 'vue';
import TranslationPanel from './components/TranslationPanel.vue';
import HistoryPanel from './components/HistoryPanel.vue';
import SettingsPanel from './components/SettingsPanel.vue';
import type {SettingsState, StatusMessage, TranslationResult, TranslationSource} from './types';
import {defaultSettingsState, formatTimestamp, mapSettings, mapTranslationResult, toSettingsPayload} from './types';
import {GetSettings, SaveSettings, StartScreenshotTranslation} from '../wailsjs/go/main/App';
import {EventsOff, EventsOn, WindowSetDarkTheme, WindowSetLightTheme, WindowSetSystemDefaultTheme} from '../wailsjs/runtime/runtime';

type ActiveTab = 'translate' | 'history' | 'settings';

const activeTab = ref<ActiveTab>('translate');
const history = ref<TranslationResult[]>([]);
const currentResult = ref<TranslationResult | null>(null);
const statusMessage = ref<StatusMessage | null>(null);
const liveTranslatedText = ref('');
const liveStreamSource = ref<TranslationSource | null>(null);
const isBusy = ref(false);
const apiKeyMissing = ref(false);
const settings = ref<SettingsState>(defaultSettingsState());
const registeredEvents = new Set<string>();

interface ToastItem {
	id: number;
	message: string;
}

const toasts = ref<ToastItem[]>([]);
let toastCounter = 0;

function pushToast(message: string, duration = 2800) {
	const id = ++toastCounter;
	toasts.value.push({id, message});
	window.setTimeout(() => {
		toasts.value = toasts.value.filter((item) => item.id !== id);
	}, duration);
}

function resetStreaming() {
	liveTranslatedText.value = '';
	liveStreamSource.value = null;
}

function applyTheme(theme: string) {
	const normalized = theme || 'system';
	document.documentElement.setAttribute('data-theme', normalized);
	try {
		switch (normalized) {
		case 'dark':
			WindowSetDarkTheme();
			break;
		case 'light':
			WindowSetLightTheme();
			break;
		default:
			WindowSetSystemDefaultTheme();
		}
	} catch (error) {
		console.warn('theme switch not supported:', error);
	}
}

function registerEvent(name: string, handler: (...payload: any[]) => void) {
	EventsOn(name, handler);
	registeredEvents.add(name);
}

function addHistoryEntry(entry: TranslationResult) {
	const existing = history.value.find((item) => item.timestamp === entry.timestamp && item.source === entry.source);
	if (!existing) {
		history.value = [entry, ...history.value].slice(0, 40);
	}
}

function handleTranslationResult(payload: any) {
	resetStreaming();
	const mapped = mapTranslationResult(payload);
	currentResult.value = mapped;
	addHistoryEntry(mapped);
	isBusy.value = false;
	statusMessage.value = {stage: 'done', message: '翻译完成'};
	activeTab.value = 'translate';
	if (settings.value.showToastOnComplete) {
		pushToast('翻译完成', 2000);
	}
}

function handleTranslationError(stage: string, message: string) {
	resetStreaming();
	isBusy.value = false;
	statusMessage.value = {stage, message};
	pushToast(message || '翻译失败');
}

async function requestScreenshot() {
	if (apiKeyMissing.value) {
		activeTab.value = 'settings';
		pushToast('请先配置 API Key');
		return;
	}
	isBusy.value = true;
	statusMessage.value = {stage: 'prepare', message: '正在等待截图区域…'};
	try {
		await StartScreenshotTranslation();
	} catch (error: any) {
		const message = error instanceof Error ? error.message : String(error);
		handleTranslationError('screenshot', message);
	}
}

async function loadSettings() {
	try {
		const dto = await GetSettings();
		settings.value = mapSettings(dto);
		applyTheme(settings.value.theme);
		apiKeyMissing.value = !settings.value.apiKeyOverride;
	} catch (error) {
		pushToast('加载配置失败');
		console.error(error);
	}
}

async function saveSettings(nextSettings: SettingsState) {
	try {
		const dto = await SaveSettings(toSettingsPayload(nextSettings));
		settings.value = mapSettings(dto);
		applyTheme(settings.value.theme);
		apiKeyMissing.value = !settings.value.apiKeyOverride;
		pushToast('设置已保存');
	} catch (error) {
		pushToast('保存设置失败');
		console.error(error);
	}
}

function handleHistorySelect(entry: TranslationResult) {
	resetStreaming();
	currentResult.value = entry;
	activeTab.value = 'translate';
}

const headerStatus = computed(() => {
	if (!statusMessage.value) {
		return '闲置';
	}
	return statusMessage.value.message;
});

	onMounted(async () => {
	registerEvent('translation:started', (payload?: Record<string, any>) => {
		resetStreaming();
		isBusy.value = true;
		const source = payload?.source || 'translation';
		statusMessage.value = {stage: source, message: source === 'screenshot' ? '等待用户选择截图区域…' : '开始翻译…'};
	});
	registerEvent('translation:progress', (payload?: Record<string, any>) => {
		const message = payload?.message || '处理中…';
		const stage = payload?.stage || 'working';
		statusMessage.value = {stage, message};
	});
	registerEvent('translation:result', (payload: any) => {
		handleTranslationResult(payload);
	});
	registerEvent('translation:error', (payload?: Record<string, any>) => {
		const stage = payload?.stage || 'error';
		const message = payload?.message || '处理失败';
		handleTranslationError(stage, message);
	});
	registerEvent('translation:delta', (payload?: Record<string, any>) => {
		const content = typeof payload?.content === 'string' ? payload.content : '';
		const source = payload?.source as TranslationSource | undefined;
		liveTranslatedText.value = content;
		if (source) {
			liveStreamSource.value = source;
		} else if (!liveStreamSource.value) {
			liveStreamSource.value = 'manual';
		}
	});
	registerEvent('translation:idle', () => {
		resetStreaming();
		isBusy.value = false;
	});
	registerEvent('translation:copied', (payload?: Record<string, any>) => {
		if (!settings.value.showToastOnComplete) {
			return;
		}
		const message = payload?.message || '翻译结果已复制';
		pushToast(message, 2200);
	});
	registerEvent('config:missing_api_key', (payload?: Record<string, any>) => {
		apiKeyMissing.value = true;
		const message = payload?.message || '未找到有效的 API Key';
		statusMessage.value = {stage: 'config', message};
		pushToast(message, 3200);
	});
	registerEvent('config:api_key_ready', () => {
		apiKeyMissing.value = false;
		pushToast('翻译服务已就绪', 2000);
	});
	registerEvent('settings:updated', (payload: any) => {
		settings.value = mapSettings(payload);
	});
	registerEvent('settings:theme', (payload?: Record<string, any>) => {
		const theme = payload?.theme || settings.value.theme;
		settings.value = {...settings.value, theme};
		applyTheme(theme);
	});

	await loadSettings();
});

onBeforeUnmount(() => {
	registeredEvents.forEach((name) => EventsOff(name));
	registeredEvents.clear();
});

const emptyHistory = computed(() => history.value.length === 0);

const lastUpdatedText = computed(() => {
	if (!currentResult.value) {
		return '';
	}
	return formatTimestamp(currentResult.value.timestamp);
});
</script>

<template>
	<div class="app-shell">
		<header class="app-header">
			<div class="title-block">
				<h1>沉浸翻译</h1>
				<p class="subtitle">快速截取、识别与翻译，专注阅读理解</p>
			</div>
			<nav class="tab-bar">
				<button :class="['tab', {active: activeTab === 'translate'}]" @click="activeTab = 'translate'">即时翻译</button>
				<button :class="['tab', {active: activeTab === 'history'}]" @click="activeTab = 'history'">翻译历史</button>
				<button :class="['tab', {active: activeTab === 'settings'}]" @click="activeTab = 'settings'">偏好设置</button>
			</nav>
			<div class="header-status">
				<span :class="['status-indicator', {busy: isBusy}]"></span>
				<span class="status-text">{{ headerStatus }}</span>
				<span v-if="lastUpdatedText" class="status-meta">最近更新：{{ lastUpdatedText }}</span>
			</div>
		</header>
		<main class="app-main">
			<TranslationPanel
				v-if="activeTab === 'translate'"
				:current-result="currentResult"
				:is-busy="isBusy"
				:status-message="statusMessage"
				:api-key-missing="apiKeyMissing"
				:streamed-text="liveTranslatedText"
				:stream-source="liveStreamSource"
				@start-screenshot="requestScreenshot"
			/>
			<HistoryPanel
				v-else-if="activeTab === 'history'"
				:history="history"
				:is-empty="emptyHistory"
				@select="handleHistorySelect"
			/>
			<SettingsPanel
				v-else
				:settings="settings"
				:api-key-missing="apiKeyMissing"
				@submit="saveSettings"
			/>
		</main>
		<footer class="app-footer">
			<div class="footer-hint">快捷提示：使用顶部按钮启动截图翻译，或在设置中开启自动复制结果。</div>
		</footer>
		<transition-group name="toast" tag="div" class="toast-stack">
			<div v-for="toast in toasts" :key="toast.id" class="toast">
				{{ toast.message }}
			</div>
		</transition-group>
	</div>
</template>

<style scoped>
.app-shell {
	display: flex;
	flex-direction: column;
	height: 100vh;
	width: 100vw;
	color: var(--color-text-primary);
}

.app-header {
	display: grid;
	grid-template-columns: auto 1fr auto;
	align-items: center;
	gap: 1.5rem;
	padding: 1.25rem 2rem 1rem;
	background: var(--surface-elevated);
	border-bottom: 1px solid var(--border-subtle);
}

.title-block h1 {
	margin: 0;
	font-size: 1.4rem;
	font-weight: 600;
}

.title-block .subtitle {
	margin: 0.25rem 0 0;
	font-size: 0.85rem;
	color: var(--color-text-secondary);
}

.tab-bar {
	display: flex;
	align-items: center;
	gap: 0.5rem;
}

.tab {
	padding: 0.45rem 1.2rem;
	border-radius: 999px;
	border: 1px solid transparent;
	background: transparent;
	color: inherit;
	font-size: 0.95rem;
	cursor: pointer;
	transition: all 0.2s ease;
}

.tab:hover {
	background: var(--surface-hover);
}

.tab.active {
	background: var(--accent);
	color: #fff;
	box-shadow: 0 6px 16px rgba(20, 131, 255, 0.25);
}

.header-status {
	display: flex;
	flex-direction: column;
	gap: 0.2rem;
	align-items: flex-end;
}

.status-indicator {
	display: inline-block;
	width: 10px;
	height: 10px;
	border-radius: 50%;
	background: var(--status-idle);
	box-shadow: 0 0 0 rgba(0, 0, 0, 0);
}

.status-indicator.busy {
	background: var(--status-busy);
	animation: pulse 1.2s infinite;
}

.status-text {
	font-size: 0.85rem;
	color: var(--color-text-secondary);
}

.status-meta {
	font-size: 0.75rem;
	color: var(--color-text-tertiary);
}

.app-main {
	flex: 1;
	padding: 1.5rem 2rem;
	background: var(--surface-base);
	overflow: auto;
}

.app-footer {
	padding: 0.75rem 2rem;
	background: var(--surface-elevated);
	border-top: 1px solid var(--border-subtle);
	color: var(--color-text-tertiary);
	font-size: 0.8rem;
}

.toast-stack {
	position: fixed;
	bottom: 1.5rem;
	right: 1.5rem;
	display: flex;
	flex-direction: column;
	gap: 0.6rem;
	z-index: 20;
}

.toast {
	min-width: 200px;
	max-width: 320px;
	padding: 0.75rem 1rem;
	background: var(--surface-toast);
	border-radius: 12px;
	box-shadow: 0 14px 28px rgba(15, 15, 30, 0.25);
	color: var(--color-text-primary);
	font-size: 0.9rem;
	border: 1px solid rgba(255, 255, 255, 0.05);
}

.toast-enter-active,
.toast-leave-active {
	transition: all 0.25s ease;
}

.toast-enter-from,
.toast-leave-to {
	opacity: 0;
	transform: translateY(12px) scale(0.98);
}

@keyframes pulse {
	0% {
		box-shadow: 0 0 0 0 rgba(20, 131, 255, 0.35);
	}
	70% {
		box-shadow: 0 0 0 8px rgba(20, 131, 255, 0);
	}
	100% {
		box-shadow: 0 0 0 0 rgba(20, 131, 255, 0);
	}
}
</style>
