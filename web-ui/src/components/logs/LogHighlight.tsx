import React, { useMemo } from 'react';

interface LogHighlightProps {
  text: string;
  search?: string;
  highlights?: Array<{
    pattern: string | RegExp;
    className: string;
    color?: string;
  }>;
  caseSensitive?: boolean;
}

export function LogHighlight({ 
  text, 
  search, 
  highlights = [], 
  caseSensitive = false 
}: LogHighlightProps) {
  const highlightedText = useMemo(() => {
    if (!search && highlights.length === 0) {
      return [{ text, highlighted: false, className: '' }];
    }

    // Combine search term and custom highlights
    const allHighlights = [...highlights];
    if (search) {
      allHighlights.unshift({
        pattern: search,
        className: 'search-highlight',
        color: '#fbbf24',
      });
    }

    // Create segments with highlights
    let segments: Array<{ text: string; highlighted: boolean; className: string; color?: string }> = [
      { text, highlighted: false, className: '', color: undefined }
    ];

    allHighlights.forEach(({ pattern, className, color }) => {
      const newSegments: typeof segments = [];
      
      segments.forEach(segment => {
        if (segment.highlighted) {
          newSegments.push(segment);
          return;
        }

        const regex = pattern instanceof RegExp 
          ? pattern 
          : new RegExp(
              pattern.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 
              caseSensitive ? 'g' : 'gi'
            );

        let lastIndex = 0;
        let match;
        const parts: typeof segments = [];

        while ((match = regex.exec(segment.text)) !== null) {
          // Add text before match
          if (match.index > lastIndex) {
            parts.push({
              text: segment.text.slice(lastIndex, match.index),
              highlighted: false,
              className: '',
              color: undefined,
            });
          }

          // Add matched text
          parts.push({
            text: match[0],
            highlighted: true,
            className,
            color,
          });

          lastIndex = match.index + match[0].length;
          
          // Prevent infinite loop with zero-width matches
          if (match[0].length === 0) {
            regex.lastIndex++;
          }
        }

        // Add remaining text
        if (lastIndex < segment.text.length) {
          parts.push({
            text: segment.text.slice(lastIndex),
            highlighted: false,
            className: '',
            color: undefined,
          });
        }

        if (parts.length > 0) {
          newSegments.push(...parts);
        } else {
          newSegments.push(segment);
        }
      });

      segments = newSegments;
    });

    return segments;
  }, [text, search, highlights, caseSensitive]);

  return (
    <>
      {highlightedText.map((segment, index) => (
        segment.highlighted ? (
          <span
            key={index}
            className={segment.className}
            style={segment.color ? { backgroundColor: segment.color } : undefined}
          >
            {segment.text}
          </span>
        ) : (
          <React.Fragment key={index}>{segment.text}</React.Fragment>
        )
      ))}
    </>
  );
}