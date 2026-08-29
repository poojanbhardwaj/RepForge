import { z } from "zod";

export const profileSchema = z.object({
  displayName: z
    .string()
    .trim()
    .min(1, "Enter a display name.")
    .refine(
      (value) => Array.from(value).length <= 80,
      "Display name must be 80 characters or fewer.",
    )
    .refine(
      (value) => !/[\u0000-\u001F\u007F]/u.test(value),
      "Display name contains unsupported characters.",
    ),
});

export type ProfileForm = z.infer<typeof profileSchema>;
