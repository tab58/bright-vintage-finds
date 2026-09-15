// The API accepts only JPEG/PNG/WebP, and iPhones hand over HEIC whenever the
// picker feels like it (an accept="" list is a hint, not a guarantee). iOS can
// decode HEIC natively, so re-encoding in a canvas here means the upload is
// always a format the API and every browser can read.
export const ACCEPTED_IMAGE_TYPES = 'image/jpeg,image/png,image/webp'

const ACCEPTED = new Set(ACCEPTED_IMAGE_TYPES.split(','))

// Long edge cap: phone photos are far larger than any listing needs, and the
// API rejects anything over 15 MB.
const MAX_EDGE = 2400

export async function toUploadableImage(file: File): Promise<File> {
  if (ACCEPTED.has(file.type.toLowerCase())) return file

  const bitmap = await createImageBitmap(file).catch(() => null)
  if (!bitmap) {
    throw new Error(`"${file.name}" is not a JPEG, PNG or WebP and this device cannot convert it`)
  }

  const scale = Math.min(1, MAX_EDGE / Math.max(bitmap.width, bitmap.height))
  const canvas = document.createElement('canvas')
  canvas.width = Math.round(bitmap.width * scale)
  canvas.height = Math.round(bitmap.height * scale)
  canvas.getContext('2d')?.drawImage(bitmap, 0, 0, canvas.width, canvas.height)
  bitmap.close()

  const blob = await new Promise<Blob | null>((resolve) =>
    canvas.toBlob(resolve, 'image/jpeg', 0.9),
  )
  if (!blob) throw new Error(`Could not convert "${file.name}" for upload`)

  return new File([blob], file.name.replace(/\.[^.]+$/, '') + '.jpg', { type: 'image/jpeg' })
}
