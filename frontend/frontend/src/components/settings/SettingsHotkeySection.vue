<script lang="ts" setup>
import {computed, ref, watch} from 'vue';
import {useSettingsForm} from './useSettingsForm';

const form = useSettingsForm();

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

const hotkeyModifiers = ref(DEFAULT_HOTKEY_MODIFIER);
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
</script>

<template>
	<div class="settings-hotkey">
		<div class="settings-hotkey__controls">
			<label class="settings-field">
				<span>修饰键组合</span>
				<select v-model="hotkeyModifiers">
					<option v-for="option in modifierOptions" :key="option.value" :value="option.value">
						{{ option.label }}
					</option>
				</select>
			</label>
			<label class="settings-field">
				<span>主触发键</span>
				<select v-model="hotkeyKey">
					<option v-for="option in keyOptions" :key="option.value" :value="option.value">
						{{ option.label }}
					</option>
				</select>
			</label>
		</div>
		<div class="settings-hotkey__preview">
			<span>热键预览：</span>
			<strong>{{ hotkeyPreview }}</strong>
		</div>
		<p class="settings-hotkey__hint">设置后可在系统范围直接唤起翻译窗口，避免与常用组合冲突。</p>
	</div>
</template>

<style scoped>
.settings-hotkey {
	display: flex;
	flex-direction: column;
	gap: 0.8rem;
}

.settings-hotkey__controls {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
	gap: 1rem;
}

.settings-field {
	display: flex;
	flex-direction: column;
	gap: 0.45rem;
	font-size: 0.9rem;
}

.settings-field span {
	font-weight: 500;
}

.settings-field select {
	background: var(--surface-base);
	border: 1px solid var(--border-subtle);
	border-radius: 12px;
	padding: 0.6rem 0.9rem;
	color: var(--color-text-primary);
	appearance: none;
	transition: border-color 0.15s ease, box-shadow 0.15s ease;
}

.settings-field select:focus {
	outline: none;
	border-color: var(--accent);
	box-shadow: 0 0 0 2px rgba(20, 131, 255, 0.25);
}

.settings-hotkey__preview {
	display: flex;
	align-items: center;
	gap: 0.3rem;
	font-size: 0.88rem;
}

.settings-hotkey__preview strong {
	font-size: 1rem;
}

.settings-hotkey__hint {
	margin: 0;
	font-size: 0.78rem;
	color: var(--color-text-tertiary);
}
</style>
