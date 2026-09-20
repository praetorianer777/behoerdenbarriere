-- Die Erklärung zur Barrierefreiheit ist keine Kennzahl, sondern eine Rechtspflicht
-- (§ 12b BGG, § 7 BITV 2.0). Sie steht deshalb neben dem Score, nicht darin.
ALTER TABLE scans ADD COLUMN statement jsonb;
