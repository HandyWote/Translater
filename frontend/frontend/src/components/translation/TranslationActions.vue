<script lang="ts" setup>
import AppButton from '../base/AppButton.vue';
import type {StatusMessage} from '../../types';

const props = defineProps<{
	isBusy: boolean;
	apiKeyMissing: boolean;
	statusMessage: StatusMessage | null;
}>();

const emit = defineEmits<{
	(event: 'start'): void;
}>();

function handleStart() {
	if (props.isBusy || props.apiKeyMissing) {
		return;
	}
	emit('start');
}
</script>

<template>
	<div class="translation-actions">
		<div class="translation-actions__row">
			<AppButton :loading="props.isBusy" :disabled="props.apiKeyMissing" @click="handleStart">
				开始截图翻译
			</AppButton>
			<span v-if="props.statusMessage" class="translation-actions__status">{{ props.statusMessage.message }}</span>
		</div>
		<p v-if="props.apiKeyMissing" class="translation-actions__warning">尚未配置 API Key，部分功能不可用，请前往偏好设置。</p>
		<p v-else class="translation-actions__hint">点击按钮启动截图翻译，结果将在下方展示并可自动复制。</p>
	</div>
</template>

<style scoped>
.translation-actions {
	display: flex;
	flex-direction: column;
	gap: 0.6rem;
	align-items: flex-start;
}

.translation-actions__row {
	display: flex;
	align-items: center;
	gap: 0.9rem;
	flex-wrap: wrap;
}

.translation-actions__status {
	font-size: 0.85rem;
	color: var(--color-text-secondary);
}

.translation-actions__warning {
	color: var(--color-warning);
	font-size: 0.85rem;
	margin: 0;
}

.translation-actions__hint {
	font-size: 0.85rem;
	color: var(--color-text-tertiary);
	margin: 0;
}
</style>
