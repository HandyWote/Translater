<script lang="ts" setup>
import {computed, ref} from 'vue';
import type {StatusMessage, TranslationResult} from '../types';
import {formatDuration} from '../types';

const props = defineProps<{
	currentResult: TranslationResult | null;
	isBusy: boolean;
	statusMessage: StatusMessage | null;
	apiKeyMissing: boolean;
}>();

const emit = defineEmits<{
	(event: 'start-screenshot'): void;
}>();

const showCopiedTip = ref(false);

function triggerScreenshot() {
	emit('start-screenshot');
}

async function copyTranslation() {
	const result = props.currentResult;
	if (!result || !result.translatedText) {
		return;
	}
	try {
		await navigator.clipboard.writeText(result.translatedText);
		showCopiedTip.value = true;
		setTimeout(() => (showCopiedTip.value = false), 1600);
	} catch (error) {
		console.warn('复制失败', error);
	}
}

const hasResult = computed(() => Boolean(props.currentResult?.translatedText?.trim()));
const durationText = computed(() => (props.currentResult ? formatDuration(props.currentResult.durationMs) : ''));
</script>

<template>
	<section class="panel">
		<div class="panel-inner">
			<div class="actions">
				<div class="action-row">
					<button class="primary" :disabled="props.isBusy || props.apiKeyMissing" @click="triggerScreenshot">
						<span v-if="props.isBusy" class="spinner"></span>
						开始截图翻译
					</button>
					<span v-if="props.statusMessage" class="inline-status">{{ props.statusMessage.message }}</span>
				</div>
				<p v-if="props.apiKeyMissing" class="warning">尚未配置 API Key，部分功能不可用，请前往偏好设置。</p>
				<p v-else class="hint">点击按钮启动截图翻译，结果将在下方展示并可自动复制。</p>
			</div>

			<div class="result-card" :class="{empty: !hasResult}">
				<header>
					<div>
						<h2>翻译结果</h2>
						<p class="meta" v-if="props.statusMessage">{{ props.statusMessage.message }}</p>
					</div>
					<div class="result-actions">
						<button class="ghost" :disabled="!hasResult" @click="copyTranslation">
							复制结果
						</button>
						<span v-if="durationText" class="duration">耗时 {{ durationText }}</span>
						<span v-if="showCopiedTip" class="copied-tip">已复制 ✓</span>
					</div>
				</header>
				<div class="result-body">
					<div v-if="hasResult" class="result-text" v-html="props.currentResult?.translatedText.replace(/\n/g, '<br />')"></div>
					<div v-else class="placeholder">等待翻译结果或从历史记录中选择。</div>
				</div>
			</div>
		</div>
	</section>
</template>

<style scoped>
.panel {
	display: flex;
	justify-content: center;
}

.panel-inner {
	width: 100%;
	max-width: 960px;
	display: flex;
	flex-direction: column;
	gap: 1.2rem;
}

.actions {
	display: flex;
	flex-direction: column;
	gap: 0.6rem;
	align-items: flex-start;
}

.action-row {
	display: flex;
	align-items: center;
	gap: 0.9rem;
	flex-wrap: wrap;
}

.inline-status {
	font-size: 0.85rem;
	color: var(--color-text-secondary);
}

button {
	border: none;
	border-radius: 10px;
	padding: 0.65rem 1.4rem;
	font-size: 0.95rem;
	cursor: pointer;
	transition: transform 0.15s ease, box-shadow 0.15s ease, background 0.15s ease;
}

button:disabled {
	opacity: 0.5;
	cursor: not-allowed;
	box-shadow: none;
}

.primary {
	background: var(--accent);
	color: #fff;
	box-shadow: 0 10px 18px rgba(20, 131, 255, 0.25);
}

.primary:hover:not(:disabled) {
	transform: translateY(-1px);
}

.ghost {
	background: transparent;
	color: var(--color-text-secondary);
	border: 1px solid var(--border-subtle);
	padding: 0.4rem 0.9rem;
	border-radius: 999px;
}

.ghost:hover:not(:disabled) {
	background: var(--surface-hover);
}

.spinner {
	margin-right: 0.4rem;
	width: 16px;
	height: 16px;
	border: 3px solid rgba(255, 255, 255, 0.3);
	border-top-color: #fff;
	border-radius: 50%;
	animation: spin 0.8s linear infinite;
}

.warning {
	color: var(--color-warning);
	font-size: 0.85rem;
}

.hint {
	font-size: 0.85rem;
	color: var(--color-text-tertiary);
}

.result-card {
	background: var(--surface-elevated);
	border-radius: 18px;
	border: 1px solid var(--border-subtle);
	box-shadow: 0 18px 32px rgba(10, 10, 30, 0.25);
	padding: 1.1rem 1.3rem;
	display: flex;
	flex-direction: column;
	gap: 0.9rem;
	min-height: 280px;
}

.result-card header {
	display: flex;
	align-items: flex-start;
	justify-content: space-between;
	gap: 0.8rem;
}

.result-card h2 {
	margin: 0;
	font-size: 1.05rem;
	font-weight: 600;
}

.result-card.empty {
	background: var(--surface-muted);
	box-shadow: none;
}

.result-actions {
	display: flex;
	align-items: center;
	gap: 0.7rem;
	flex-wrap: wrap;
}

.result-body {
	flex: 1;
	min-height: 220px;
	background: var(--surface-base);
	border-radius: 12px;
	border: 1px dashed var(--border-subtle);
	padding: 1rem 1.2rem;
	overflow-y: auto;
}

.result-text {
	white-space: pre-wrap;
	line-height: 1.6;
	color: var(--color-text-primary);
}

.placeholder {
	color: var(--color-text-tertiary);
	font-size: 0.92rem;
	text-align: center;
	margin-top: 1.6rem;
}

.meta {
	margin: 0.3rem 0 0;
	font-size: 0.82rem;
	color: var(--color-text-tertiary);
}

.duration {
	font-size: 0.75rem;
	color: var(--color-text-tertiary);
}

.copied-tip {
	font-size: 0.75rem;
	color: var(--accent);
	font-weight: 600;
}

@media (max-width: 1000px) {
	.panel-inner {
		max-width: 100%;
	}

	.result-body {
		min-height: 180px;
	}
}

@keyframes spin {
	to {
		transform: rotate(360deg);
	}
}
</style>
