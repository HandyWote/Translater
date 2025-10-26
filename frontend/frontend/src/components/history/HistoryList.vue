<script lang="ts" setup>
import type {TranslationResult} from '../../types';
import {formatDuration, formatTimestamp} from '../../types';

const props = defineProps<{
	items: TranslationResult[];
}>();

const emit = defineEmits<{
	(event: 'select', value: TranslationResult): void;
}>();

function handleSelect(entry: TranslationResult) {
	emit('select', entry);
}
</script>

<template>
	<ul class="history-list">
		<li v-for="item in props.items" :key="item.timestamp + item.source" @click="handleSelect(item)">
			<div class="history-list__meta">
				<span class="history-list__source" :data-source="item.source">{{ item.source === 'screenshot' ? '截图' : '手动' }}</span>
				<span class="history-list__time">{{ formatTimestamp(item.timestamp) }}</span>
				<span v-if="item.durationMs" class="history-list__duration">{{ formatDuration(item.durationMs) }}</span>
			</div>
			<div class="history-list__preview">
				<p class="history-list__original" :title="item.originalText">{{ item.originalText || '（空文本）' }}</p>
				<p class="history-list__translated" :title="item.translatedText">{{ item.translatedText || '无翻译结果' }}</p>
			</div>
		</li>
	</ul>
</template>

<style scoped>
.history-list {
	list-style: none;
	margin: 0;
	padding: 0;
	display: flex;
	flex-direction: column;
	gap: 0.8rem;
}

.history-list li {
	padding: 0.9rem 1.1rem;
	border-radius: 16px;
	background: var(--surface-elevated);
	border: 1px solid transparent;
	box-shadow: 0 12px 24px rgba(10, 10, 30, 0.18);
	cursor: pointer;
	transition: transform 0.18s ease, box-shadow 0.18s ease, border-color 0.18s ease;
}

.history-list li:hover {
	transform: translateY(-2px);
	box-shadow: 0 18px 34px rgba(15, 15, 30, 0.22);
	border-color: var(--border-subtle);
}

.history-list__meta {
	display: flex;
	gap: 0.8rem;
	align-items: center;
	font-size: 0.82rem;
	color: var(--color-text-tertiary);
}

.history-list__source {
	padding: 0.1rem 0.6rem;
	border-radius: 999px;
	background: var(--surface-hover);
	text-transform: uppercase;
	font-size: 0.7rem;
	letter-spacing: 0.08em;
}

.history-list__source[data-source='screenshot'] {
	background: rgba(20, 131, 255, 0.18);
	color: #5bb1ff;
}

.history-list__source[data-source='manual'] {
	background: rgba(90, 245, 195, 0.16);
	color: #63d3a9;
}

.history-list__preview {
	margin-top: 0.55rem;
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
	gap: 0.8rem;
}

.history-list__preview p {
	margin: 0;
	font-size: 0.92rem;
	line-height: 1.45;
	max-height: 4.2rem;
	overflow: hidden;
	text-overflow: ellipsis;
}

.history-list__original {
	color: var(--color-text-secondary);
}

.history-list__translated {
	color: var(--color-text-primary);
	font-weight: 500;
}

.history-list__duration {
	font-size: 0.75rem;
}

@media (max-width: 720px) {
	.history-list__preview {
		grid-template-columns: 1fr;
	}
}
</style>
