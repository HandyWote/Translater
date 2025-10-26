<script lang="ts" setup>
import PanelShell from './base/PanelShell.vue';
import HistoryEmptyState from './history/HistoryEmptyState.vue';
import HistoryList from './history/HistoryList.vue';
import type {TranslationResult} from '../types';

const props = defineProps<{
	history: TranslationResult[];
	isEmpty: boolean;
}>();

const emit = defineEmits<{
	(event: 'select', value: TranslationResult): void;
}>();

function handleSelect(entry: TranslationResult) {
	emit('select', entry);
}
</script>

<template>
	<PanelShell>
		<header class="history-header">
			<h2>翻译历史</h2>
		</header>
		<HistoryEmptyState v-if="props.isEmpty" />
		<HistoryList v-else :items="props.history" @select="handleSelect" />
	</PanelShell>
</template>

<style scoped>
.history-header {
	display: flex;
	flex-direction: column;
	gap: 0.35rem;
	color: var(--color-text-primary);
}

.history-header h2 {
	margin: 0;
	font-size: 1.2rem;
	font-weight: 600;
}

.history-header p {
	margin: 0;
	color: var(--color-text-tertiary);
	font-size: 0.9rem;
}
</style>
