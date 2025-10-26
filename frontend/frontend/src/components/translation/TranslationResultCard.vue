<script lang="ts" setup>
import {computed} from 'vue';
import AppButton from '../base/AppButton.vue';
import {useClipboard} from '../../composables/useClipboard';
import type {StatusMessage, TranslationSource} from '../../types';

const props = defineProps<{
	text: string;
	statusMessage: StatusMessage | null;
	durationText: string;
	streamSource: TranslationSource | null;
	isStreaming: boolean;
}>();

const {copy, copied, copying} = useClipboard();

const normalizedText = computed(() => props.text?.trim() ?? '');
const hasResult = computed(() => normalizedText.value.length > 0);
const formattedText = computed(() => normalizedText.value.replace(/\n/g, '<br />'));
const streamSourceLabel = computed(() => {
	switch (props.streamSource) {
	case 'screenshot':
		return '截图';
	case 'manual':
		return '手动';
	default:
		return '';
	}
});

function copyResult() {
	if (!hasResult.value) {
		return;
	}
	copy(normalizedText.value);
}
</script>

<template>
	<div :class="['translation-card', {'translation-card--empty': !hasResult}]">
		<header class="translation-card__header">
			<div>
				<h2>翻译结果</h2>
				<p v-if="props.statusMessage" class="translation-card__meta">{{ props.statusMessage.message }}</p>
			</div>
			<div class="translation-card__actions">
				<span v-if="props.isStreaming" class="translation-card__live">
					实时输出中<span v-if="streamSourceLabel">（{{ streamSourceLabel }}）</span>…
				</span>
				<AppButton
					variant="ghost"
					:disabled="!hasResult"
					:loading="copying"
					@click="copyResult"
				>
					复制结果
				</AppButton>
				<span v-if="props.durationText" class="translation-card__duration">耗时 {{ props.durationText }}</span>
				<span v-if="copied" class="translation-card__copied">已复制 ✓</span>
			</div>
		</header>
		<div class="translation-card__body">
			<div v-if="hasResult" class="translation-card__text" v-html="formattedText"></div>
			<div v-else class="translation-card__placeholder">等待翻译结果或从历史记录中选择。</div>
		</div>
	</div>
</template>

<style scoped>
.translation-card {
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

.translation-card--empty {
	background: var(--surface-muted);
	box-shadow: none;
}

.translation-card__header {
	display: flex;
	align-items: flex-start;
	justify-content: space-between;
	gap: 0.8rem;
}

.translation-card__header h2 {
	margin: 0;
	font-size: 1.05rem;
	font-weight: 600;
}

.translation-card__meta {
	margin: 0.3rem 0 0;
	font-size: 0.82rem;
	color: var(--color-text-tertiary);
}

.translation-card__actions {
	display: flex;
	align-items: center;
	gap: 0.7rem;
	flex-wrap: wrap;
}

.translation-card__live {
	font-size: 0.78rem;
	color: var(--accent);
	font-weight: 600;
	letter-spacing: 0.02em;
}

.translation-card__body {
	flex: 1;
	min-height: 220px;
	background: var(--surface-base);
	border-radius: 12px;
	border: 1px dashed var(--border-subtle);
	padding: 1rem 1.2rem;
	overflow-y: auto;
}

.translation-card__text {
	white-space: pre-wrap;
	line-height: 1.6;
	color: var(--color-text-primary);
}

.translation-card__placeholder {
	color: var(--color-text-tertiary);
	font-size: 0.92rem;
	text-align: center;
	margin-top: 1.6rem;
}

.translation-card__duration,
.translation-card__copied {
	font-size: 0.75rem;
	color: var(--color-text-tertiary);
}

.translation-card__copied {
	color: var(--accent);
	font-weight: 600;
}

@media (max-width: 1000px) {
	.translation-card__body {
		min-height: 180px;
	}
}
</style>
