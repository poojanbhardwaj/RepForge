import { profileSchema } from "./schema";

const supplementaryCharacter = "\u{1F3CB}";

it("accepts 80 supplementary Unicode characters", () => {
  const displayName = supplementaryCharacter.repeat(80);

  expect(profileSchema.parse({ displayName })).toEqual({ displayName });
});

it("rejects 81 supplementary Unicode characters", () => {
  const result = profileSchema.safeParse({ displayName: supplementaryCharacter.repeat(81) });

  expect(result.success).toBe(false);
  if (!result.success) {
    expect(result.error.issues[0]?.message).toBe("Display name must be 80 characters or fewer.");
  }
});

it("counts mixed ASCII and supplementary characters by Unicode code point", () => {
  expect(
    profileSchema.safeParse({ displayName: `A${supplementaryCharacter}`.repeat(40) }).success,
  ).toBe(true);
  expect(
    profileSchema.safeParse({ displayName: `A${supplementaryCharacter}`.repeat(40) + "B" }).success,
  ).toBe(false);
});

it("preserves the normal ASCII boundary", () => {
  expect(profileSchema.safeParse({ displayName: "A".repeat(80) }).success).toBe(true);
  expect(profileSchema.safeParse({ displayName: "A".repeat(81) }).success).toBe(false);
});
