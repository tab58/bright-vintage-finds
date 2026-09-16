// Every picture is re-encoded here before upload, for two reasons. Phone
// photos are enormous — a 12 MP iPhone shot is ~4.5 MB, far more than a
// listing needs — and iPhones hand over HEIC whenever the picker feels like it
// (an accept="" list is a hint, not a guarantee), which the API rejects. One
// canvas pass solves both: the upload is always a reasonably sized JPEG.
export const ACCEPTED_IMAGE_TYPES = 'image/jpeg,image/png,image/webp';

const ACCEPTED = new Set(ACCEPTED_IMAGE_TYPES.split(','));

// Size knobs. 1600px / 0.82 turns that 4.5 MB photo into ~0.58 MB, which is
// still sharper than the shop front or the lightbox ever renders. Raise
// MAX_EDGE for more detail, raise QUALITY for less JPEG mush; both cost bytes.
const MAX_EDGE = 1600;
const QUALITY = 0.82;

export async function toUploadableImage(file: File): Promise<File> {
  const accepted = ACCEPTED.has(file.type.toLowerCase());

  const bitmap = await createImageBitmap(file).catch(() => null);
  if (!bitmap) {
    // Nothing decodable here. If the API takes this type as-is, let it through
    // at full size rather than failing the upload outright.
    if (accepted) return file;
    throw new Error(
      `"${file.name}" is not a JPEG, PNG or WebP and this device cannot convert it`,
    );
  }

  const scale = Math.min(1, MAX_EDGE / Math.max(bitmap.width, bitmap.height));

  // Already small enough and in a format the API takes: send the original
  // bytes instead of re-encoding and losing quality for nothing.
  if (scale === 1 && accepted) {
    bitmap.close();
    return file;
  }

  const canvas = document.createElement('canvas');
  canvas.width = Math.round(bitmap.width * scale);
  canvas.height = Math.round(bitmap.height * scale);
  const ctx = canvas.getContext('2d');
  // JPEG has no alpha, so a transparent PNG would otherwise flatten to black.
  if (ctx) {
    ctx.fillStyle = '#fff';
    ctx.fillRect(0, 0, canvas.width, canvas.height);
    ctx.drawImage(bitmap, 0, 0, canvas.width, canvas.height);
  }
  bitmap.close();

  const blob = await new Promise<Blob | null>((resolve) =>
    canvas.toBlob(resolve, 'image/jpeg', QUALITY),
  );
  if (!blob) throw new Error(`Could not convert "${file.name}" for upload`);

  return new File([blob], file.name.replace(/\.[^.]+$/, '') + '.jpg', {
    type: 'image/jpeg',
  });
}
