import {SelectDirectory} from '../wailsjs/go/main/App'

export type DirectoryPicker = () => Promise<string>

export const nativeDirectoryPicker: DirectoryPicker = async () => {
	const selected = await SelectDirectory()
	return typeof selected === 'string' ? selected : ''
}
