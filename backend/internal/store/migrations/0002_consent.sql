CREATE TYPE consent_state AS ENUM ('none', 'declined', 'accepted', 'blocked');

-- Whether a consent layer stood in the way decides what the score is about. Without
-- this column a page checked behind a banner looks like any other page.
ALTER TABLE pages ADD COLUMN consent consent_state NOT NULL DEFAULT 'none';

-- How far a scan actually reached: pages that stayed behind a banner describe the
-- banner, not the site, and a reader has to be able to see how many those were.
ALTER TABLE scans ADD COLUMN pages_blocked integer NOT NULL DEFAULT 0;
