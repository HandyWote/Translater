<script lang="ts" setup>
import {computed} from 'vue';
import PanelShell from './base/PanelShell.vue';
import TranslationActions from './translation/TranslationActions.vue';
import TranslationResultCard from './translation/TranslationResultCard.vue';
import type {StatusMessage, TranslationResult, TranslationSource} from '../types';
import {formatDuration} from '../types';

const props = defineProps<{
	currentResult: TranslationResult | null;
	isBusy: boolean;
	statusMessage: StatusMessage | null;
	apiKeyMissing: boolean;
	streamedText: string;
	streamSource: TranslationSource | null;
}>();

const emit = defineEmits<{
	(event: 'start-screenshot'): void;
}>();

const displayText = computed(() => {
	const finalText = props.currentResult?.translatedText ?? '';
	if (finalText.trim()) {
		return finalText;
	}
	const streamed = props.streamedText?.trim();
	if (streamed) {
		return props.streamedText;
	}
	return '';
});

const streamingActive = computed(() => {
	if (props.currentResult?.translatedText?.trim()) {
		return false;
	}
	return Boolean(props.streamedText?.trim());
});
const durationText = computed(() => (props.currentResult ? formatDuration(props.currentResult.durationMs) : ''));

function handleStart() {
	emit('start-screenshot');
}
</script>

<template>
	<PanelShell>
		<TranslationActions
			:is-busy="props.isBusy"
			:api-key-missing="props.apiKeyMissing"
			:status-message="props.statusMessage"
			@start="handleStart"
		/>
		<TranslationResultCard
			:text="displayText"
			:status-message="props.statusMessage"
			:duration-text="durationText"
			:stream-source="props.streamSource"
			:is-streaming="streamingActive"
		/>
	</PanelShell>
</template>
