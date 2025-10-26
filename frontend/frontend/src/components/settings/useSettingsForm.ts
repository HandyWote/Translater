import {inject, provide} from 'vue';
import type {SettingsState} from '../../types';

const settingsFormKey = Symbol('settings-form');

export function provideSettingsForm(form: SettingsState) {
	provide(settingsFormKey, form);
}

export function useSettingsForm() {
	const form = inject<SettingsState>(settingsFormKey);
	if (!form) {
		throw new Error('Settings form context  未提供');
	}
	return form;
}
