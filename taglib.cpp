//go:build ignore
#include <cstdint>
#include <cstring>
#include <iostream>

#include "fileref.h"
#include "tpropertymap.h"
#include "mpeg/mpegfile.h"
#include "mpeg/id3v1/id3v1tag.h"
#include "mpeg/id3v2/id3v2tag.h"
#include "mpeg/id3v2/frames/textidentificationframe.h"
#include "mpeg/id3v2/frames/commentsframe.h"
#include "mpeg/id3v2/frames/popularimeterframe.h"
#include "mpeg/id3v2/frames/unsynchronizedlyricsframe.h"
#include "mpeg/id3v2/frames/synchronizedlyricsframe.h"
#include "mp4/mp4file.h"
#include "mp4/mp4tag.h"
#include "mp4/mp4item.h"
#include "flac/flacfile.h"
#include "flac/flacproperties.h"
#include "mp4/mp4properties.h"
#include "riff/aiff/aifffile.h"
#include "riff/aiff/aiffproperties.h"
#include "riff/wav/wavfile.h"
#include "riff/wav/wavproperties.h"
#include "ape/apeproperties.h"
#include "asf/asffile.h"
#include "asf/asfproperties.h"
#include "asf/asftag.h"
#include "asf/asfattribute.h"
#include "wavpack/wavpackproperties.h"
#include "dsf/dsfproperties.h"

char *to_char_array(const TagLib::String &s) {
  const std::string str = s.to8Bit(true);
  return ::strdup(str.c_str());
}

TagLib::String to_string(const char *s) {
  return TagLib::String(s, TagLib::String::UTF8);
}

__attribute__((export_name("malloc"))) void *exported_malloc(size_t size) {
  return malloc(size);
}

__attribute__((export_name("taglib_file_tags"))) char **
taglib_file_tags(const char *filename) {
  TagLib::FileRef file(filename);
  if (file.isNull())
    return nullptr;

  auto properties = file.properties();

  size_t len = 0;
  for (const auto &kvs : properties)
    len += kvs.second.size();

  char **tags = static_cast<char **>(malloc(sizeof(char *) * (len + 1)));
  if (!tags)
    return nullptr;

  size_t i = 0;
  for (const auto &kvs : properties)
    for (const auto &v : kvs.second) {
      TagLib::String row = kvs.first + "\t" + v;
      tags[i] = to_char_array(row);
      i++;
    }
  tags[len] = nullptr;

  return tags;
}

static const uint8_t CLEAR = 1 << 0;

__attribute__((export_name("taglib_file_write_tags"))) bool
taglib_file_write_tags(const char *filename, const char **tags, uint8_t opts) {
  if (!filename || !tags)
    return false;

  TagLib::FileRef file(filename);
  if (file.isNull())
    return false;

  auto properties = file.properties();
  if (opts & CLEAR)
    properties.clear();

  for (size_t i = 0; tags[i]; i++) {
    TagLib::String row(tags[i], TagLib::String::UTF8);
    if (auto ti = row.find("\t"); ti != -1) {
      auto key = row.substr(0, ti);
      auto value = row.substr(ti + 1);
      if (value.isEmpty())
        properties.erase(key);
      else
        properties.replace(key, value.split("\v"));
    }
  }

  file.setProperties(properties);
  return file.save();
}

struct FileProperties {
  uint32_t lengthInMilliseconds;
  uint32_t channels;
  uint32_t sampleRate;
  uint32_t bitrate;
  uint32_t bitsPerSample;
  char **imageMetadata;
};

__attribute__((export_name("taglib_file_read_properties"))) FileProperties *
taglib_file_read_properties(const char *filename) {
  TagLib::FileRef file(filename);
  if (file.isNull() || !file.audioProperties())
    return nullptr;

  FileProperties *props =
      static_cast<FileProperties *>(malloc(sizeof(FileProperties)));
  if (!props)
    return nullptr;

  auto audioProperties = file.audioProperties();
  props->lengthInMilliseconds = audioProperties->lengthInMilliseconds();
  props->channels = audioProperties->channels();
  props->sampleRate = audioProperties->sampleRate();
  props->bitrate = audioProperties->bitrate();

  // Extract bits per sample for supported formats
  int bitsPerSample = 0;
  if (const auto* apeProperties = dynamic_cast<const TagLib::APE::Properties*>(audioProperties))
    bitsPerSample = apeProperties->bitsPerSample();
  else if (const auto* asfProperties = dynamic_cast<const TagLib::ASF::Properties*>(audioProperties))
    bitsPerSample = asfProperties->bitsPerSample();
  else if (const auto* flacProperties = dynamic_cast<const TagLib::FLAC::Properties*>(audioProperties))
    bitsPerSample = flacProperties->bitsPerSample();
  else if (const auto* mp4Properties = dynamic_cast<const TagLib::MP4::Properties*>(audioProperties))
    bitsPerSample = mp4Properties->bitsPerSample();
  else if (const auto* wavPackProperties = dynamic_cast<const TagLib::WavPack::Properties*>(audioProperties))
    bitsPerSample = wavPackProperties->bitsPerSample();
  else if (const auto* aiffProperties = dynamic_cast<const TagLib::RIFF::AIFF::Properties*>(audioProperties))
    bitsPerSample = aiffProperties->bitsPerSample();
  else if (const auto* wavProperties = dynamic_cast<const TagLib::RIFF::WAV::Properties*>(audioProperties))
    bitsPerSample = wavProperties->bitsPerSample();
  else if (const auto* dsfProperties = dynamic_cast<const TagLib::DSF::Properties*>(audioProperties))
    bitsPerSample = dsfProperties->bitsPerSample();
  props->bitsPerSample = bitsPerSample > 0 ? bitsPerSample : 0;

  const auto &pictures = file.complexProperties("PICTURE");

  props->imageMetadata = nullptr;
  if (pictures.isEmpty())
    return props;

  size_t len = pictures.size();
  char **imageMetadata =
      static_cast<char **>(malloc(sizeof(char *) * (len + 1)));
  if (!imageMetadata)
    return props;

  size_t i = 0;
  for (const auto &p : pictures) {
    TagLib::String type = p["pictureType"].toString();
    TagLib::String desc = p["description"].toString();
    TagLib::String mime = p["mimeType"].toString();
    TagLib::String row = type + "\t" + desc + "\t" + mime;
    imageMetadata[i] = to_char_array(row);
    i++;
  }
  imageMetadata[len] = nullptr;

  props->imageMetadata = imageMetadata;

  return props;
}

struct ByteData {
  uint32_t length;
  char *data;
};

__attribute__((export_name("taglib_file_read_image"))) ByteData *
taglib_file_read_image(const char *filename, int index) {
  TagLib::FileRef file(filename);
  if (file.isNull())
    return nullptr;

  const auto &pictures = file.complexProperties("PICTURE");
  if (pictures.isEmpty())
    return nullptr;

  if (index < 0 || index >= static_cast<int>(pictures.size()))
    return nullptr;

  auto v = pictures[index]["data"].toByteVector();
  ByteData *bd = static_cast<ByteData *>(malloc(sizeof(ByteData)));
  if (!bd)
    return nullptr;

  bd->length = static_cast<uint32_t>(v.size());
  if (bd->length == 0) {
    bd->data = nullptr;
    return bd;
  }

  // allocate and copy into module memory to keep it valid for go to read
  char *buf = static_cast<char *>(malloc(bd->length));
  if (!buf)
    return nullptr;

  memcpy(buf, v.data(), bd->length);
  bd->data = buf;

  return bd;
}

__attribute__((export_name("taglib_file_write_image"))) bool
taglib_file_write_image(const char *filename, const char *buf, uint32_t length,
                        int index, const char *pictureType,
                        const char *description, const char *mimeType) {
  TagLib::FileRef file(filename);
  if (file.isNull())
    return false;

  auto pictures = file.complexProperties("PICTURE");

  if (length == 0) {
    // remove image at index if it exists
    if (index >= 0 && index < static_cast<int>(pictures.size())) {
      auto it = pictures.begin();
      std::advance(it, index);
      pictures.erase(it);
      if (!file.setComplexProperties("PICTURE", pictures))
        return false;
    }
    return file.save();
  }

  TagLib::VariantMap newPicture;
  newPicture["data"] = TagLib::ByteVector(buf, length);
  newPicture["pictureType"] = to_string(pictureType);
  newPicture["description"] = to_string(description);
  newPicture["mimeType"] = to_string(mimeType);

  // replace image at index, or append if index is out of range
  if (index >= 0 && index < static_cast<int>(pictures.size()))
    pictures[index] = newPicture;
  else
    pictures.append(newPicture);

  if (!file.setComplexProperties("PICTURE", pictures))
    return false;

  return file.save();
}

__attribute__((export_name("taglib_file_id3v2_frames"))) char **
taglib_file_id3v2_frames(const char *filename) {
  // First check if this is an MP3 file with ID3v2 tags
  TagLib::FileRef fileRef(filename);
  if (fileRef.isNull())
    return nullptr;

  // Try to cast to MPEG::File
  TagLib::MPEG::File *mpegFile = dynamic_cast<TagLib::MPEG::File *>(fileRef.file());
  if (!mpegFile || !mpegFile->hasID3v2Tag()) {
    // Return empty array instead of nullptr when there are no ID3v2 tags
    char **emptyFrames = static_cast<char **>(malloc(sizeof(char *)));
    if (!emptyFrames)
      return nullptr;
    emptyFrames[0] = nullptr;
    return emptyFrames;
  }

  TagLib::ID3v2::Tag *id3v2Tag = mpegFile->ID3v2Tag();
  const TagLib::ID3v2::FrameListMap &frameListMap = id3v2Tag->frameListMap();

  // Count total number of frames
  size_t frameCount = 0;
  for (TagLib::ID3v2::FrameListMap::ConstIterator it = frameListMap.begin(); it != frameListMap.end(); ++it) {
    frameCount += it->second.size();
  }

  if (frameCount == 0) {
    // Return empty array if there are no frames
    char **emptyFrames = static_cast<char **>(malloc(sizeof(char *)));
    if (!emptyFrames)
      return nullptr;
    emptyFrames[0] = nullptr;
    return emptyFrames;
  }

  // Allocate result array
  char **frames = static_cast<char **>(malloc(sizeof(char *) * (frameCount + 1)));
  if (!frames)
    return nullptr;

  size_t i = 0;

  // Process each frame
  for (TagLib::ID3v2::FrameListMap::ConstIterator it = frameListMap.begin(); it != frameListMap.end(); ++it) {
    TagLib::String frameID = TagLib::String(it->first);

    for (TagLib::ID3v2::FrameList::ConstIterator frameIt = it->second.begin(); frameIt != it->second.end(); ++frameIt) {
      TagLib::String key = frameID;
      TagLib::String value;

      // Handle special frame types
      if (frameID == "TXXX") {
        // User text identification frame
        auto userFrame = dynamic_cast<TagLib::ID3v2::UserTextIdentificationFrame *>(*frameIt);
        if (userFrame) {
          key = frameID + ":" + userFrame->description();
          if (!userFrame->fieldList().isEmpty()) {
            value = userFrame->fieldList().back();
          }
        }
      }
      else if (frameID == "COMM") {
        // Comments frame
        auto commFrame = dynamic_cast<TagLib::ID3v2::CommentsFrame *>(*frameIt);
        if (commFrame) {
          key = frameID + ":" + commFrame->description();
          value = commFrame->text();
        }
      }
      else if (frameID == "POPM") {
        // Popularimeter frame (used for WMP ratings)
        auto popmFrame = dynamic_cast<TagLib::ID3v2::PopularimeterFrame *>(*frameIt);
        if (popmFrame) {
          key = frameID + ":" + popmFrame->email();
          value = TagLib::String::number(popmFrame->rating());
        }
      }
      else if (frameID == "USLT") {
        // Unsynchronized lyrics frame
        auto usltFrame = dynamic_cast<TagLib::ID3v2::UnsynchronizedLyricsFrame *>(*frameIt);
        if (usltFrame) {
          // Get language code (3 characters, e.g., "eng", "xxx")
          TagLib::ByteVector lang = usltFrame->language();
          TagLib::String langStr = "xxx";
          if (lang.size() == 3) {
            char langBuf[4] = {0};
            memcpy(langBuf, lang.data(), 3);
            langStr = TagLib::String(langBuf);
          }
          key = frameID + ":" + langStr;
          value = usltFrame->text();
        }
      }
      else if (frameID == "SYLT") {
        // Synchronized lyrics frame - convert to LRC format
        auto syltFrame = dynamic_cast<TagLib::ID3v2::SynchronizedLyricsFrame *>(*frameIt);
        if (syltFrame) {
          // Get language code (3 characters)
          TagLib::ByteVector lang = syltFrame->language();
          TagLib::String langStr = "xxx";
          if (lang.size() == 3) {
            char langBuf[4] = {0};
            memcpy(langBuf, lang.data(), 3);
            langStr = TagLib::String(langBuf);
          }
          key = frameID + ":" + langStr;

          // Build LRC format from synchronized text
          TagLib::String lrc;
          auto format = syltFrame->timestampFormat();
          for (const auto &syncText : syltFrame->synchedText()) {
            int timeMs = syncText.time;
            if (format == TagLib::ID3v2::SynchronizedLyricsFrame::AbsoluteMpegFrames) {
              // Skip MPEG frames format - would need sample rate to convert
              continue;
            }
            int mins = timeMs / 60000;
            int secs = (timeMs % 60000) / 1000;
            int centis = (timeMs % 1000) / 10;
            char timeBuf[16];
            snprintf(timeBuf, sizeof(timeBuf), "[%02d:%02d.%02d]", mins, secs, centis);
            lrc = lrc + TagLib::String(timeBuf) + syncText.text + "\n";
          }
          value = lrc;
        }
      }
      else {
        // Standard frame
        value = (*frameIt)->toString();
      }

      // Create the output string
      TagLib::String row = key + "\t" + value;
      frames[i++] = to_char_array(row);
    }
  }

  frames[i] = nullptr;
  return frames;
}

__attribute__((export_name("taglib_file_id3v1_tags"))) char **
taglib_file_id3v1_tags(const char *filename) {
  // First check if this is an MP3 file with ID3v1 tags
  TagLib::FileRef fileRef(filename);
  if (fileRef.isNull())
    return nullptr;

  // Try to cast to MPEG::File
  TagLib::MPEG::File *mpegFile = dynamic_cast<TagLib::MPEG::File *>(fileRef.file());
  if (!mpegFile || !mpegFile->hasID3v1Tag()) {
    // Return empty array instead of nullptr when there are no ID3v1 tags
    char **emptyTags = static_cast<char **>(malloc(sizeof(char *)));
    if (!emptyTags)
      return nullptr;
    emptyTags[0] = nullptr;
    return emptyTags;
  }

  TagLib::ID3v1::Tag *id3v1Tag = mpegFile->ID3v1Tag();

  // ID3v1 has a fixed set of fields
  const int fieldCount = 7; // title, artist, album, year, comment, track, genre
  char **tags = static_cast<char **>(malloc(sizeof(char *) * (fieldCount + 1)));
  if (!tags)
    return nullptr;

  int i = 0;

  // Add each standard ID3v1 field
  if (!id3v1Tag->title().isEmpty())
    tags[i++] = to_char_array(TagLib::String("TITLE\t") + id3v1Tag->title());

  if (!id3v1Tag->artist().isEmpty())
    tags[i++] = to_char_array(TagLib::String("ARTIST\t") + id3v1Tag->artist());

  if (!id3v1Tag->album().isEmpty())
    tags[i++] = to_char_array(TagLib::String("ALBUM\t") + id3v1Tag->album());

  // Year is an unsigned int in ID3v1, convert to string
  if (id3v1Tag->year() > 0)
    tags[i++] = to_char_array(TagLib::String("YEAR\t") + TagLib::String::number(id3v1Tag->year()));

  if (!id3v1Tag->comment().isEmpty())
    tags[i++] = to_char_array(TagLib::String("COMMENT\t") + id3v1Tag->comment());

  if (id3v1Tag->track() > 0)
    tags[i++] = to_char_array(TagLib::String("TRACK\t") + TagLib::String::number(id3v1Tag->track()));

  // Genre is an int in ID3v1, need to get the string representation
  if (id3v1Tag->genreNumber() != 255) { // 255 is used for "unknown genre"
    if (!id3v1Tag->genre().isEmpty())
      tags[i++] = to_char_array(TagLib::String("GENRE\t") + id3v1Tag->genre());
  }

  tags[i] = nullptr;
  return tags;
}

__attribute__((export_name("taglib_file_mp4_atoms"))) char **
taglib_file_mp4_atoms(const char *filename) {
  TagLib::FileRef fileRef(filename);
  if (fileRef.isNull())
    return nullptr;

  // Try to cast to MP4::File
  TagLib::MP4::File *mp4File = dynamic_cast<TagLib::MP4::File *>(fileRef.file());
  if (!mp4File || !mp4File->hasMP4Tag()) {
    // Return empty array instead of nullptr when there are no MP4 atoms
    char **emptyAtoms = static_cast<char **>(malloc(sizeof(char *)));
    if (!emptyAtoms)
      return nullptr;
    emptyAtoms[0] = nullptr;
    return emptyAtoms;
  }

  TagLib::MP4::Tag *mp4Tag = mp4File->tag();
  const TagLib::MP4::ItemMap &itemMap = mp4Tag->itemMap();

  // First pass: count total entries (multi-value items count as multiple)
  size_t atomCount = 0;
  for (auto it = itemMap.begin(); it != itemMap.end(); ++it) {
    TagLib::MP4::Item item = it->second;
    switch (item.type()) {
      case TagLib::MP4::Item::Type::StringList:
        atomCount += item.toStringList().size();
        break;
      case TagLib::MP4::Item::Type::IntPair:
        atomCount += 2; // num and total as separate keys
        break;
      default:
        atomCount++;
        break;
    }
  }

  if (atomCount == 0) {
    char **emptyAtoms = static_cast<char **>(malloc(sizeof(char *)));
    if (!emptyAtoms)
      return nullptr;
    emptyAtoms[0] = nullptr;
    return emptyAtoms;
  }

  char **atoms = static_cast<char **>(malloc(sizeof(char *) * (atomCount + 1)));
  if (!atoms)
    return nullptr;

  size_t i = 0;
  for (auto it = itemMap.begin(); it != itemMap.end(); ++it) {
    TagLib::String key = it->first;
    TagLib::MP4::Item item = it->second;

    switch (item.type()) {
      case TagLib::MP4::Item::Type::Bool: {
        TagLib::String value = item.toBool() ? "1" : "0";
        TagLib::String row = key + "\t" + value;
        atoms[i++] = to_char_array(row);
        break;
      }
      case TagLib::MP4::Item::Type::Int: {
        TagLib::String value = TagLib::String::number(item.toInt());
        TagLib::String row = key + "\t" + value;
        atoms[i++] = to_char_array(row);
        break;
      }
      case TagLib::MP4::Item::Type::IntPair: {
        auto pair = item.toIntPair();
        TagLib::String numRow = key + ":num\t" + TagLib::String::number(pair.first);
        TagLib::String totalRow = key + ":total\t" + TagLib::String::number(pair.second);
        atoms[i++] = to_char_array(numRow);
        atoms[i++] = to_char_array(totalRow);
        break;
      }
      case TagLib::MP4::Item::Type::Byte: {
        TagLib::String value = TagLib::String::number(item.toByte());
        TagLib::String row = key + "\t" + value;
        atoms[i++] = to_char_array(row);
        break;
      }
      case TagLib::MP4::Item::Type::UInt: {
        TagLib::String value = TagLib::String::number(item.toUInt());
        TagLib::String row = key + "\t" + value;
        atoms[i++] = to_char_array(row);
        break;
      }
      case TagLib::MP4::Item::Type::LongLong: {
        TagLib::String value = TagLib::String::number(item.toLongLong());
        TagLib::String row = key + "\t" + value;
        atoms[i++] = to_char_array(row);
        break;
      }
      case TagLib::MP4::Item::Type::StringList: {
        TagLib::StringList sl = item.toStringList();
        for (const auto &s : sl) {
          TagLib::String row = key + "\t" + s;
          atoms[i++] = to_char_array(row);
        }
        break;
      }
      case TagLib::MP4::Item::Type::CoverArtList:
      case TagLib::MP4::Item::Type::ByteVectorList: {
        // Include binary data atoms with empty value (like ID3v2 does for APIC)
        TagLib::String row = key + "\t";
        atoms[i++] = to_char_array(row);
        break;
      }
      default:
        break;
    }
  }

  atoms[i] = nullptr;
  return atoms;
}

__attribute__((export_name("taglib_file_asf_attributes"))) char **
taglib_file_asf_attributes(const char *filename) {
  TagLib::FileRef fileRef(filename);
  if (fileRef.isNull())
    return nullptr;

  // Try to cast to ASF::File
  TagLib::ASF::File *asfFile = dynamic_cast<TagLib::ASF::File *>(fileRef.file());
  if (!asfFile || !asfFile->tag()) {
    // Return empty array instead of nullptr when there are no ASF attributes
    char **emptyAttrs = static_cast<char **>(malloc(sizeof(char *)));
    if (!emptyAttrs)
      return nullptr;
    emptyAttrs[0] = nullptr;
    return emptyAttrs;
  }

  TagLib::ASF::Tag *asfTag = asfFile->tag();
  const TagLib::ASF::AttributeListMap &attrMap = asfTag->attributeListMap();

  // Count basic fields (Title, Author, Copyright, Description, Rating)
  size_t basicCount = 0;
  if (!asfTag->title().isEmpty()) basicCount++;
  if (!asfTag->artist().isEmpty()) basicCount++;
  if (!asfTag->copyright().isEmpty()) basicCount++;
  if (!asfTag->comment().isEmpty()) basicCount++;
  if (!asfTag->rating().isEmpty()) basicCount++;

  // Count total entries (multi-value attributes count as multiple)
  size_t attrCount = basicCount;
  for (auto it = attrMap.begin(); it != attrMap.end(); ++it) {
    attrCount += it->second.size();
  }

  if (attrCount == 0) {
    char **emptyAttrs = static_cast<char **>(malloc(sizeof(char *)));
    if (!emptyAttrs)
      return nullptr;
    emptyAttrs[0] = nullptr;
    return emptyAttrs;
  }

  char **attrs = static_cast<char **>(malloc(sizeof(char *) * (attrCount + 1)));
  if (!attrs)
    return nullptr;

  size_t i = 0;

  // Add basic fields first (these are stored separately from the attributeListMap)
  if (!asfTag->title().isEmpty()) {
    TagLib::String row = TagLib::String("Title\t") + asfTag->title();
    attrs[i++] = to_char_array(row);
  }
  if (!asfTag->artist().isEmpty()) {
    TagLib::String row = TagLib::String("Author\t") + asfTag->artist();
    attrs[i++] = to_char_array(row);
  }
  if (!asfTag->copyright().isEmpty()) {
    TagLib::String row = TagLib::String("Copyright\t") + asfTag->copyright();
    attrs[i++] = to_char_array(row);
  }
  if (!asfTag->comment().isEmpty()) {
    TagLib::String row = TagLib::String("Description\t") + asfTag->comment();
    attrs[i++] = to_char_array(row);
  }
  if (!asfTag->rating().isEmpty()) {
    TagLib::String row = TagLib::String("Rating\t") + asfTag->rating();
    attrs[i++] = to_char_array(row);
  }

  // Add extended attributes
  for (auto it = attrMap.begin(); it != attrMap.end(); ++it) {
    TagLib::String key = it->first;
    const TagLib::ASF::AttributeList &attrList = it->second;

    for (const auto &attr : attrList) {
      TagLib::String value;
      switch (attr.type()) {
        case TagLib::ASF::Attribute::UnicodeType:
          value = attr.toString();
          break;
        case TagLib::ASF::Attribute::BoolType:
          value = attr.toBool() ? "1" : "0";
          break;
        case TagLib::ASF::Attribute::DWordType:
          value = TagLib::String::number(attr.toUInt());
          break;
        case TagLib::ASF::Attribute::QWordType:
          value = TagLib::String::number(static_cast<long long>(attr.toULongLong()));
          break;
        case TagLib::ASF::Attribute::WordType:
          value = TagLib::String::number(attr.toUShort());
          break;
        case TagLib::ASF::Attribute::BytesType:
        case TagLib::ASF::Attribute::GuidType:
          // Binary data - include with empty value (like ID3v2 does for APIC)
          value = "";
          break;
        default:
          continue;
      }

      TagLib::String row = key + "\t" + value;
      attrs[i++] = to_char_array(row);
    }
  }

  attrs[i] = nullptr;
  return attrs;
}

__attribute__((export_name("taglib_file_write_id3v2_frames"))) bool
taglib_file_write_id3v2_frames(const char *filename, const char **frames, uint8_t opts) {
  if (!filename || !frames)
    return false;

  // First check if this is an MP3 file with ID3v2 tags
  TagLib::MPEG::File file(filename);
  if (!file.isValid())
    return false;

  // Create a new ID3v2 tag if one doesn't exist
  if (!file.hasID3v2Tag()) {
    file.ID3v2Tag(true);
  }

  TagLib::ID3v2::Tag *id3v2Tag = file.ID3v2Tag();

  // If clear option is set, collect all frame IDs we want to keep
  bool clearFrames = (opts & CLEAR);

  // First collect all the frame IDs we're going to set
  std::vector<TagLib::ByteVector> frameIDsToKeep;
  if (clearFrames) {
    for (int i = 0; frames[i] != nullptr; i++) {
      TagLib::String row(frames[i], TagLib::String::UTF8);
      int ti = row.find("\t");
      if (ti != -1) {
        TagLib::String key = row.substr(0, ti);
        // Store the base frame ID (without description for TXXX, COMM, etc.)
        if (key.find(":") != -1) {
          key = key.substr(0, key.find(":"));
        }
        frameIDsToKeep.push_back(key.data(TagLib::String::Latin1));
      }
    }

    // Now remove all frames except those we're going to set
    const TagLib::ID3v2::FrameListMap &frameListMap = id3v2Tag->frameListMap();
    for (TagLib::ID3v2::FrameListMap::ConstIterator it = frameListMap.begin();
         it != frameListMap.end(); ++it) {
      bool keepFrame = false;
      for (size_t i = 0; i < frameIDsToKeep.size(); ++i) {
        if (it->first == frameIDsToKeep[i]) {
          keepFrame = true;
          break;
        }
      }
      if (!keepFrame) {
        id3v2Tag->removeFrames(it->first);
      }
    }
  }

  // Now add the new frames
  for (int i = 0; frames[i] != nullptr; i++) {
    TagLib::String row(frames[i], TagLib::String::UTF8);
    int ti = row.find("\t");
    if (ti != -1) {
      TagLib::String key = row.substr(0, ti);
      TagLib::String value = row.substr(ti + 1);

      // Remove existing frames with this ID
      id3v2Tag->removeFrames(key.toCString(true));

      // Add new frame if value is not empty
      if (!value.isEmpty()) {
        if (key.startsWith("T")) {
          // Text identification frame
          auto newFrame = new TagLib::ID3v2::TextIdentificationFrame(key.toCString(true), TagLib::String::UTF8);
          TagLib::StringList values;

          // Split value by vertical tab
          int pos = 0;
          while (pos != -1) {
            int nextPos = value.find("\v", pos);
            if (nextPos == -1) {
              values.append(value.substr(pos));
              break;
            } else {
              values.append(value.substr(pos, nextPos - pos));
              pos = nextPos + 1;
            }
          }

          newFrame->setText(values);
          id3v2Tag->addFrame(newFrame);
        }
        else if (key == "COMM") {
          // Comments frame
          auto newFrame = new TagLib::ID3v2::CommentsFrame(TagLib::String::UTF8);
          newFrame->setText(value);
          id3v2Tag->addFrame(newFrame);
        }
        // Add other frame types as needed
      }
    }
  }

  // Save the file
  return file.save();
}
