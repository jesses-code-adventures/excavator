CREATE TABLE IF NOT EXISTS User (
    id INTEGER PRIMARY KEY,
    name TEXT UNIQUE NOT NULL
);

INSERT INTO User (name) VALUES ('jesse');

CREATE TABLE IF NOT EXISTS Collection (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    FOREIGN KEY (user_id) REFERENCES User(id)
);

CREATE TABLE IF NOT EXISTS Tag (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL,
    file_path TEXT UNIQUE NOT NULL,
    FOREIGN KEY (user_id) REFERENCES User(id)
);

CREATE TABLE IF NOT EXISTS CollectionTag (
    id INTEGER PRIMARY KEY,
    tag_id INTEGER NOT NULL,
    collection_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    sub_collection TEXT,
    FOREIGN KEY (tag_id) REFERENCES Tag(id),
    FOREIGN KEY (collection_id) REFERENCES Collection(id)
);

CREATE TABLE IF NOT EXISTS Export (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    output_dir TEXT NOT NULL,
    FOREIGN KEY (user_id) REFERENCES User(id)
);

CREATE TABLE IF NOT EXISTS ExportTag (
    id INTEGER PRIMARY KEY,
    collection_id INTEGER NOT NULL,
    export_id INTEGER NOT NULL,
    FOREIGN KEY (collection_id) REFERENCES Collection(id),
    FOREIGN KEY (export_id) REFERENCES Export(id)
);
