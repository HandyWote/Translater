<script lang="ts" setup>
import {computed, ref, watch} from 'vue';
import {
	DEFAULT_EXTRACT_PROMPT,
	DEFAULT_TRANSLATE_PROMPT,
	defaultSettingsState,
} from '../../types';
import {useSettingsForm} from './useSettingsForm';

const form = useSettingsForm();

const enableCustomPrompts = ref(
	form.extractPrompt !== DEFAULT_EXTRACT_PROMPT || form.translatePrompt !== DEFAULT_TRANSLATE_PROMPT,
);

let syncingPrompts = false;

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

watch(
	[() => form.extractPrompt, () => form.translatePrompt],
	([extract, translate]) => {
		if (syncingPrompts) {
			return;
		}
		enableCustomPrompts.value = !(extract === DEFAULT_EXTRACT_PROMPT && translate === DEFAULT_TRANSLATE_PROMPT);
	},
);

watch(enableCustomPrompts, (enabled) => {
	if (enabled) {
		return;
	}
	syncingPrompts = true;
	form.extractPrompt = DEFAULT_EXTRACT_PROMPT;
	form.translatePrompt = DEFAULT_TRANSLATE_PROMPT;
	syncingPrompts = false;
});

function resetPrompts() {
	const defaults = defaultSettingsState();
	form.extractPrompt = defaults.extractPrompt;
	form.translatePrompt = defaults.translatePrompt;
	enableCustomPrompts.value = false;
}
</script>

<template>
	<div class="settings-prompts">
		<label class="settings-toggle">
			<input v-model="enableCustomPrompts" type="checkbox" />
			<div>
				<strong>启用自定义提示词</strong>
				<span>覆盖默认策略，适配专有术语与行业语境。</span>
			</div>
		</label>
		<div v-if="enableCustomPrompts" class="prompt-fields">
			<label class="prompt-field">
				<span>视觉识别提示词</span>
				<textarea v-model="extractPromptField" rows="4" placeholder="默认策略" />
			</label>
			<label class="prompt-field">
				<span>文本翻译提示词</span>
				<textarea v-model="translatePromptField" rows="4" placeholder="默认策略" />
			</label>
			<div class="prompt-actions">
				<button type="button" class="prompt-reset" @click="resetPrompts">恢复默认</button>
			</div>
		</div>
	</div>
</template>

<style scoped>
.settings-prompts {
	display: flex;
	flex-direction: column;
	gap: 0.9rem;
}

.settings-toggle {
	display: flex;
	gap: 0.75rem;
	align-items: flex-start;
	padding: 0.7rem 0.85rem;
	border-radius: 12px;
	background: var(--surface-base);
	border: 1px solid var(--border-subtle);
	cursor: pointer;
}

.settings-toggle input {
	margin-top: 0.3rem;
}

.settings-toggle strong {
	font-size: 0.92rem;
	font-weight: 600;
}

.settings-toggle span {
	display: block;
	margin-top: 0.2rem;
	color: var(--color-text-tertiary);
	font-size: 0.78rem;
	line-height: 1.35;
}

.prompt-fields {
	display: flex;
	flex-direction: column;
	gap: 0.9rem;
}

.prompt-field {
	display: flex;
	flex-direction: column;
	gap: 0.45rem;
}

.prompt-field span {
	font-weight: 500;
}

.prompt-field textarea {
	background: var(--surface-base);
	border: 1px solid var(--border-subtle);
	border-radius: 12px;
	color: var(--color-text-primary);
	padding: 0.75rem 0.9rem;
	font-size: 0.86rem;
	min-height: 140px;
	resize: vertical;
	transition: border-color 0.15s ease, box-shadow 0.15s ease;
}

.prompt-field textarea:focus {
	outline: none;
	border-color: var(--accent);
	box-shadow: 0 0 0 2px rgba(20, 131, 255, 0.2);
}

.prompt-actions {
	display: flex;
	justify-content: flex-end;
}

.prompt-reset {
	background: transparent;
	border: 1px dashed var(--border-subtle);
	border-radius: 999px;
	padding: 0.4rem 1rem;
	color: var(--color-text-secondary);
	cursor: pointer;
	transition: background 0.15s ease, border-color 0.15s ease;
}

.prompt-reset:hover {
	background: var(--surface-hover);
	border-color: var(--border-subtle);
}
</style>
